package skewBinomialQ

import (
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

// TODO I'm a golang newb, make more elegant decorators
var cachedMaxParallelism *int

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
	}
	*(lazyQ.pendingOpsCount) = 0
	return lazyQ
}

func (q LazyMergeSkewBinomialQueue) IncrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, 1)
}

func (q LazyMergeSkewBinomialQueue) DecrOpsCount() {
	atomic.AddInt32(q.pendingOpsCount, -1)
}

func (q LazyMergeSkewBinomialQueue) blockUntilNoPending() {
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
	q.startMeldFreeQueues()
	return q
}

func (q LazyMergeSkewBinomialQueue) startMeldFreeQueues() {
	q.IncrOpsCount()
	go q.meldFreeQueues()
}

func (q LazyMergeSkewBinomialQueue) meldFreeQueues() {
	defer q.DecrOpsCount()
	queuesToFetch := 2
	if !q.freeQueueList.LengthGreaterThan(MaxParallelism() + (queuesToFetch - 1)) {
		return
	}

	var queues []unsafe.Pointer
	failAddress := new(int8)
	counter := 0
	for len(queues) < queuesToFetch {
		poppedQ := q.freeQueueList.PopNth(MaxParallelism(), unsafe.Pointer(failAddress))
		counter++
		// return current list of queues into the list
		if (*int8)(poppedQ) == failAddress {
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
		qPtr := otherList.PopFirst()
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
	// TODO...welp...this thing is just wrong
	length := 0
	for qPtr := range q.freeQueueList.Iter() {
		priorityQ := (*BootstrappedSkewBinomialQueue)(qPtr)
		length += priorityQ.Length()
	}
	return length
}

func (q LazyMergeSkewBinomialQueue) IsEmpty() bool {
	firstQPtr := q.freeQueueList.Peek()
	if firstQPtr == nil {
		return true
	}
	return (*SkewBinomialQueue)(firstQPtr).IsEmpty()
}

func (q LazyMergeSkewBinomialQueue) Dequeue() (QueuePriority, PriorityQueue) {
	panic("NOT REDY YET")
	var qPtr unsafe.Pointer
	for {
		qPtr = q.freeQueueList.PopFirst()
		if qPtr == nil && *(q.pendingOpsCount) > 0 {
			// explicitly yield to another goroutine
			time.Sleep(0)
		} else {
			break
		}
	}
	bootstrappedQ := (*BootstrappedSkewBinomialQueue)(qPtr)
	if bootstrappedQ == nil {
		return nil, q
	}
	if bootstrappedQ.IsEmpty() {
		if q.freeQueueList.LengthGreaterThan(0) {
			return q.Dequeue()
		}
		if *(q.pendingOpsCount) > 0 {
			time.Sleep(0) // explicit yield
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

	q.IncrOpsCount()
	go q.asyncInsert(remainingBootstrappedQ)
	return queuePriority, q
}

func (q LazyMergeSkewBinomialQueue) asyncInsert(bootstrappedQ BootstrappedSkewBinomialQueue) {
	defer q.DecrOpsCount()
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q LazyMergeSkewBinomialQueue) dequeueCallback(childNodes []Node, remainingQueues ...*SkewBinomialQueue) SkewBinomialQueue {
	panic("ALSO NOT YET")
	q.IncrOpsCount()
	go q.asyncDequeueCallback(childNodes, remainingQueues[1:]...)
	return *(remainingQueues[0])
}

func (q LazyMergeSkewBinomialQueue) transformAndInsert(skewQ SkewBinomialQueue) {
	defer q.DecrOpsCount()
	bootstrappedQ := skewQToBootstrappedQ(skewQ)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q LazyMergeSkewBinomialQueue) startTransformAndInsert(skewQ SkewBinomialQueue) {
	panic("NOT YET")
	q.IncrOpsCount()
	go q.transformAndInsert(skewQ)
}

func (q LazyMergeSkewBinomialQueue) asyncDequeueCallback(childNodes []Node, remainingQueues ...*SkewBinomialQueue) {
	defer q.DecrOpsCount()
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
