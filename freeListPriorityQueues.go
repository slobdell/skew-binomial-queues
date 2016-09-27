package skewBinomialQ

import (
	"bytes"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

// TODO I'm a golang newb, make more elegant decorators
var cachedMaxParallelism *int

var FAIL_ADDRESS unsafe.Pointer = unsafe.Pointer(new(int8))

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

func skewQToBootstrappedQ(q SkewBinomialQueue) BootstrappedSkewBinomialQueue {
	panic("use the new function I wrote like a badass")
}

func qLessThanOther(q1 unsafe.Pointer, q2 unsafe.Pointer) bool {
	priorityQ1 := (*BootstrappedSkewBinomialQueue)(q1)
	priorityQ2 := (*BootstrappedSkewBinomialQueue)(q2)
	if priorityQ2.IsEmpty() {
		return false
	}

	if priorityQ1.IsEmpty() {
		return false
	} else if priorityQ2.IsEmpty() {
		return true
	} else {
		return priorityQ1.Peek().LessThan(priorityQ2.Peek())
	}
}

type LazyMergeSkewBinomialQueue struct {
	freeQueueList   *ThreadSafeList
	pendingOpsCount *int32
	length          *int64
}

func NewEmptyLazyMergeSkewBinomialQueue() PriorityQueue {
	primitiveQ := NewEmptyBootstrappedSkewBinomialQueue()
	threadSafeList := ThreadSafeList{}
	threadSafeList.InsertObject(
		unsafe.Pointer(&primitiveQ),
		qLessThanOther,
	)

	lazyQ := LazyMergeSkewBinomialQueue{
		freeQueueList:   &threadSafeList,
		pendingOpsCount: new(int32),
		length:          new(int64),
	}
	*(lazyQ.pendingOpsCount) = 0
	return lazyQ
}

func (q LazyMergeSkewBinomialQueue) incrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, 1)
}

func (q LazyMergeSkewBinomialQueue) decrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, -1)
}

func (q LazyMergeSkewBinomialQueue) incrLength() {
	atomic.AddInt64(q.length, 1)
}

func (q LazyMergeSkewBinomialQueue) decrLength() {
	atomic.AddInt64(q.length, -1)
}

func (q LazyMergeSkewBinomialQueue) BlockUntilNoPending() {
	for {
		currentValue := *(q.pendingOpsCount)
		if currentValue == 0 {
			break
		}
		time.Sleep(0)
	}
}

func (q LazyMergeSkewBinomialQueue) Enqueue(priority QueuePriority) PriorityQueue {
	sizeOneQ := NewEmptyBootstrappedSkewBinomialQueue().Enqueue(priority)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&sizeOneQ),
		qLessThanOther,
	)
	q.incrLength()
	q.startMeldFreeQueues()
	return q
}

func (q LazyMergeSkewBinomialQueue) startMeldFreeQueues() {
	q.incrOpsCount()
	go q.meldFreeQueues()
}

func (q LazyMergeSkewBinomialQueue) meldFreeQueues() {
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
	q1 := *((*BootstrappedSkewBinomialQueue)(queues[0]))
	q2 := *((*BootstrappedSkewBinomialQueue)(queues[1]))
	finalQ := (q1.Meld(q2))
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&finalQ),
		qLessThanOther,
	)
	if q.freeQueueList.LengthGreaterThan(MaxParallelism()) {
		q.startMeldFreeQueues()
	}
}

func (q LazyMergeSkewBinomialQueue) Peek() QueuePriority {
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
	firstQ := (*BootstrappedSkewBinomialQueue)(qPtr)
	return firstQ.Peek()
}

func (q LazyMergeSkewBinomialQueue) Meld(otherQ PriorityQueue) PriorityQueue {
	panic("TEMP")
	otherLazyQ := otherQ.(LazyMergeSkewBinomialQueue)
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
	go q.meldFreeQueues()
	return q
}

func (q LazyMergeSkewBinomialQueue) Length() int {
	return int(*(q.length))
}

func (q LazyMergeSkewBinomialQueue) IsEmpty() bool {
	firstQPtr := q.freeQueueList.Peek()
	if firstQPtr == nil {
		return true
	}
	return (*SkewBinomialQueue)(firstQPtr).IsEmpty()
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

func (q LazyMergeSkewBinomialQueue) Dequeue() (QueuePriority, PriorityQueue) {
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

	bootstrappedQ := (*BootstrappedSkewBinomialQueue)(qPtr)

	if bootstrappedQ.IsEmpty() {
		if q.Length() > 0 {
			return q.Dequeue()
		}
		return nil, q
	}
	/*
		queuePriority, remainingBootstrappedQ := bootstrappedQ.DequeueWithCallback(
			// TODO FIX THIS SHIT
			//q.dequeueCallback,
			bootstrappedQ.priorityQueue.dequeueCallback,
		)
	*/
	queuePriority, remainingBootstrappedQ := bootstrappedQ.Dequeue()

	q.startInsert(remainingBootstrappedQ)
	defer q.decrLength()
	return queuePriority, q
}

func (q LazyMergeSkewBinomialQueue) startInsert(bootstrappedQ BootstrappedSkewBinomialQueue) {
	q.incrOpsCount()
	go q.asyncInsert(bootstrappedQ)
}

func (q LazyMergeSkewBinomialQueue) asyncInsert(bootstrappedQ BootstrappedSkewBinomialQueue) {
	defer q.decrOpsCount()
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q LazyMergeSkewBinomialQueue) dequeueCallback(childNodes []Node, remainingQueues ...*SkewBinomialQueue) SkewBinomialQueue {
	panic("ALSO NOT YET")
	q.incrOpsCount()
	go q.asyncDequeueCallback(childNodes, remainingQueues[1:]...)
	return *(remainingQueues[0])
}

func (q LazyMergeSkewBinomialQueue) transformAndInsert(skewQ SkewBinomialQueue) {
	defer q.decrOpsCount()
	bootstrappedQ := skewQToBootstrappedQ(skewQ)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q LazyMergeSkewBinomialQueue) startTransformAndInsert(skewQ SkewBinomialQueue) {
	panic("NOT YET")
	q.incrOpsCount()
	go q.transformAndInsert(skewQ)
}

func (q LazyMergeSkewBinomialQueue) asyncDequeueCallback(childNodes []Node, remainingQueues ...*SkewBinomialQueue) {
	defer q.decrOpsCount()
	panic("NOT READY FOR THIS YET")
	for _, skewQ := range remainingQueues {
		q.startTransformAndInsert(*skewQ)
	}

	var prioritiesRankZero []QueuePriority
	for _, child := range childNodes {
		if child.Rank() > 0 {
			validQ := newSkewBinomialQueue(
				child,
				nil,
			)
			q.startTransformAndInsert(validQ)
		} else {
			prioritiesRankZero = append(
				prioritiesRankZero,
				child.Peek(),
			)
		}
	}
	lastQ := NewEmptySkewBinomialQueue().bulkInsert(prioritiesRankZero...)
	q.startTransformAndInsert(lastQ)
	q.startMeldFreeQueues()
}
