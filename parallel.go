package priorityq

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// this number is based off of simple tuning experimentation
const PARALLELISM_COEFF = 2

// TODO I'm a golang newb, make more elegant decorators
var cachedMaxParallelism *int

var FAIL_ADDRESS unsafe.Pointer = unsafe.Pointer(new(int8))
var meldSignaler = make(chan ParallelQ)
var once sync.Once

type insertBootstrappedParams struct {
	q            ParallelQ
	bootstrapped bootstrappedSkewBinomial
}

type insertSkewParams struct {
	q     ParallelQ
	skewQ skewBinomial
}

var insertSignaler = make(chan insertBootstrappedParams)
var insertSkewSignaler = make(chan insertSkewParams)

const MELD_PERIOD = 2

func MaxParallelism() int {
	if cachedMaxParallelism != nil {
		return *cachedMaxParallelism
	}
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()

	var returnValue int
	if maxProcs < numCPU {
		returnValue = maxProcs
	}
	returnValue = numCPU
	if cachedMaxParallelism == nil {
		cachedMaxParallelism = new(int)
		*cachedMaxParallelism = returnValue
	}
	return returnValue
}

func meldWorker() {
	for {
		q := <-meldSignaler
		q.meldFreeQueues()
	}
}

func asyncInsertWorker() {
	for {
		params := <-insertSignaler
		params.q.asyncInsert(params.bootstrapped)
	}
}

func asyncInsertSkewWorker() {
	for {
		params := <-insertSkewSignaler
		params.q.asyncInsertSkew(params.skewQ)
	}
}

func qLessThanOther(q1 unsafe.Pointer, q2 unsafe.Pointer) bool {
	priorityQ1 := (*bootstrappedSkewBinomial)(q1)
	priorityQ2 := (*bootstrappedSkewBinomial)(q2)
	if priorityQ2.IsEmpty() {
		return false
	}

	if priorityQ1.IsEmpty() {
		return false
	} else if priorityQ2.IsEmpty() {
		return true
	} else {
		return priorityQ1.Peek().Score() < priorityQ2.Peek().Score()
	}
}

type ParallelQ struct {
	freeQueueList   *ThreadSafeList
	pendingOpsCount *int32
	length          *int64
	meldCounter     *int32
}

func (q ParallelQ) incrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, 1)
}

func (q ParallelQ) decrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, -1)
}

func (q ParallelQ) incrLength() {
	atomic.AddInt64(q.length, 1)
}

func (q ParallelQ) decrLength() {
	atomic.AddInt64(q.length, -1)
}

func (q ParallelQ) BlockUntilNoPending() {
	for {
		currentValue := *(q.pendingOpsCount)
		if currentValue == 0 {
			break
		}
		time.Sleep(0)
	}
}

func (q ParallelQ) Enqueue(priority PriorityScorer) PriorityQ {
	sizeOneQ := newEmptyBootstrappedSkewBinomial().Enqueue(priority)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&sizeOneQ),
		qLessThanOther,
	)
	q.incrLength()
	q.startMeldFreeQueues()
	return q
}

func (q ParallelQ) startMeldFreeQueues() {
	if atomic.AddInt32(q.meldCounter, 1)%MELD_PERIOD != 0 {
		return
	}
	q.incrOpsCount()
	meldSignaler <- q
}

func (q ParallelQ) meldFreeQueues() {
	defer q.decrOpsCount()
	queuesToFetch := 2
	if !q.freeQueueList.LengthGreaterThan(MaxParallelism() + (queuesToFetch - 1)) {
		return
	}

	var queues []unsafe.Pointer
	counter := 0
	for len(queues) < queuesToFetch {
		poppedQ := q.freeQueueList.PopNth(MaxParallelism(), unsafe.Pointer(FAIL_ADDRESS))
		counter++
		// return current list of queues into the list
		if poppedQ == FAIL_ADDRESS {
			for _, queuePtr := range queues {
				q.freeQueueList.InsertObject(queuePtr, qLessThanOther)
			}
			return
		}
		queues = append(queues, poppedQ)
		time.Sleep(0)
	}
	q1 := *((*bootstrappedSkewBinomial)(queues[0]))
	q2 := *((*bootstrappedSkewBinomial)(queues[1]))
	finalQ := (q1.Meld(q2))
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&finalQ),
		qLessThanOther,
	)
	if q.freeQueueList.LengthGreaterThan(MaxParallelism()) {
		q.startMeldFreeQueues()
	}
}

func (q ParallelQ) Peek() PriorityScorer {
	// TODO unsure if this piece is valid...
	// TODO not actually valid..
	var qPtr unsafe.Pointer
	for {
		qPtr = q.freeQueueList.Peek()
		if qPtr == nil {
			// explicitly yield to another goroutine
			time.Sleep(0)
		}
		break
	}
	firstQ := (*bootstrappedSkewBinomial)(qPtr)
	return firstQ.Peek()
}

func (q ParallelQ) Meld(otherQ PriorityQ) PriorityQ {
	panic("do not use until we have test coverage")
	otherLazyQ := otherQ.(ParallelQ)
	otherList := otherLazyQ.freeQueueList
	for {
		qPtr := otherList.PopFirst(FAIL_ADDRESS)
		if qPtr == nil {
			break
		}
		q.freeQueueList.InsertObject(
			qPtr,
			qLessThanOther,
		)
	}
	meldSignaler <- q
	return q
}

func (q ParallelQ) Length() int {
	return int(*(q.length))
}

func (q ParallelQ) IsEmpty() bool {
	firstQPtr := q.freeQueueList.Peek()
	if firstQPtr == nil {
		return true
	}
	return (*skewBinomial)(firstQPtr).IsEmpty()
}

func getGID() uint64 {
	/*
		For debugging only! Delete once finished
	*/
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func (q ParallelQ) Dequeue() (PriorityScorer, PriorityQ) {
	// fmt.Printf("current count of list: %d\n", q.freeQueueList.Count())
	var qPtr unsafe.Pointer = FAIL_ADDRESS

	for {
		qPtr = q.freeQueueList.PopFirst(FAIL_ADDRESS)
		if qPtr == FAIL_ADDRESS {
			if q.Length() == 0 {
				return nil, q
			} else {
				time.Sleep(0)
			}
		} else {
			break
		}
	}

	bootstrappedQ := (*bootstrappedSkewBinomial)(qPtr)

	if bootstrappedQ.IsEmpty() {
		if q.Length() > 0 {
			return q.Dequeue()
		}
		return nil, q
	}
	queuePriority, remainingBootstrappedQ := bootstrappedQ.DequeueWithMergeCallback(
		q.lazyMergeCallback,
	)

	q.startInsert(remainingBootstrappedQ)
	defer q.decrLength()
	return queuePriority, q
}

func (q ParallelQ) lazyMergeCallback(childNodes []node, remainingQueues ...*skewBinomial) skewBinomial {
	passThruQueuePtr := remainingQueues[0]
	passThruQ := *passThruQueuePtr
	var validSkewQs []skewBinomial = make([]skewBinomial, 0, len(remainingQueues)-1+len(childNodes))

	for _, skewQ := range remainingQueues[1:] {
		newlyAllocatedItem := skewQ
		validSkewQs = append(validSkewQs, *newlyAllocatedItem)
	}
	var prioritiesRankZero []PriorityScorer
	for _, child := range childNodes {
		if child.Rank() > 0 {
			validQ := newSkewBinomial(
				child,
				nil,
			)
			validSkewQs = append(validSkewQs, validQ)
		} else {
			prioritiesRankZero = append(
				prioritiesRankZero,
				child.Peek(),
			)
		}
	}
	freshQ := newEmptySkewBinomial().bulkInsert(prioritiesRankZero...)
	q.startInsertSkew(freshQ)
	for _, skewQ := range validSkewQs {
		q.startInsertSkew(skewQ)
	}
	q.startMeldFreeQueues()
	return passThruQ
}

func (q ParallelQ) startInsert(bootstrappedQ bootstrappedSkewBinomial) {
	q.incrOpsCount()
	insertSignaler <- insertBootstrappedParams{
		q,
		bootstrappedQ,
	}
}

func (q ParallelQ) startInsertSkew(skewQ skewBinomial) {
	q.incrOpsCount()
	insertSkewSignaler <- insertSkewParams{
		q,
		skewQ,
	}
}

func (q ParallelQ) asyncInsertSkew(skewQ skewBinomial) {
	q.asyncInsert(
		skewQToBootstrappedQ(skewQ),
	)
}

func (q ParallelQ) asyncInsert(bootstrappedQ bootstrappedSkewBinomial) {
	defer q.decrOpsCount()
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q ParallelQ) transformAndInsert(skewQ skewBinomial) {
	defer q.decrOpsCount()
	bootstrappedQ := skewQToBootstrappedQ(skewQ)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func newParallelQ() PriorityQ {
	once.Do(func() {
		for i := 0; i < int(PARALLELISM_COEFF*float32(MaxParallelism())); i++ {
			go meldWorker()
			go asyncInsertWorker()
			go asyncInsertSkewWorker()
		}
	})
	primitiveQ := newEmptyBootstrappedSkewBinomial()
	threadSafeList := ThreadSafeList{}
	threadSafeList.InsertObject(
		unsafe.Pointer(&primitiveQ),
		qLessThanOther,
	)

	lazyQ := ParallelQ{
		freeQueueList:   &threadSafeList,
		pendingOpsCount: new(int32),
		length:          new(int64),
		meldCounter:     new(int32),
	}
	return lazyQ
}
