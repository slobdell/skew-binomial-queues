package skewBinomialQ

import (
	"fmt"
	"time"
	"unsafe"
)

const MIN_QUEUE_SIZE = 2

var taskSignaler chan int64 = make(chan int64)

func skewQToBootstrappedQ(q SkewBinomialQueue) BootstrappedSkewBinomialQueue {
	queuePriority, remainingQ := q.Dequeue()
	return BootstrappedSkewBinomialQueue{
		highestPriorityObject: queuePriority,
		priorityQueue:         remainingQ.(SkewBinomialQueue),
		length:                q.Length(),
	}
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
	freeQueueList    *ThreadSafeList
	pendingTaskCount *int64
}

func NewEmptyLazyMergeSkewBinomialQueue() PriorityQueue {
	primitiveQ := NewEmptyBootstrappedSkewBinomialQueue()
	threadSafeList := ThreadSafeList{}
	threadSafeList.InsertObject(
		unsafe.Pointer(&primitiveQ),
		qLessThanOther,
	)

	lazyQ := LazyMergeSkewBinomialQueue{
		freeQueueList:    &threadSafeList,
		pendingTaskCount: new(int64),
	}
	return lazyQ
}

func (q LazyMergeSkewBinomialQueue) countTasks() {
	for {
		select {
		case taskCount := <-taskSignaler:
			*(q.pendingTaskCount) += taskCount
			break
		}
	}
}

func (q LazyMergeSkewBinomialQueue) Enqueue(priority QueuePriority) PriorityQueue {
	sizeOneQ := NewEmptyBootstrappedSkewBinomialQueue().Enqueue(priority)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&sizeOneQ),
		qLessThanOther,
	)
	go q.meldFreeQueues()
	return q
}

func (q LazyMergeSkewBinomialQueue) meldFreeQueues() {
	if !q.freeQueueList.LengthGreaterThan(MIN_QUEUE_SIZE) {
		return
	}
	queuesToFetch := 2
	var queues []unsafe.Pointer = q.freeQueueList.TrySplitLastN(
		TrySplitParameter{
			N: queuesToFetch,
		},
	)
	if len(queues) < queuesToFetch {
		for _, queuePtr := range queues {
			q.freeQueueList.InsertObject(
				queuePtr,
				qLessThanOther,
			)
		}
	} else {
		q1 := *((*BootstrappedSkewBinomialQueue)(queues[0]))
		q2 := *((*BootstrappedSkewBinomialQueue)(queues[1]))
		finalQ := (q1.Meld(q2))
		q.freeQueueList.InsertObject(
			unsafe.Pointer(&finalQ),
			qLessThanOther,
		)
		if q.freeQueueList.LengthGreaterThan(MIN_QUEUE_SIZE) {
			go q.meldFreeQueues()
		}
	}
}

func (q LazyMergeSkewBinomialQueue) Peek() QueuePriority {
	// TODO unsure if this piece is valid...
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
	var qPtr unsafe.Pointer
	for {
		qPtr = q.freeQueueList.PopFirst()
		if qPtr == nil {
			fmt.Printf("GOT HERE, DETECTED NIL\n")
			// explicitly yield to another goroutine
			time.Sleep(0)
		} else {
			break
		}
	}
	bootstrappedQ := (*BootstrappedSkewBinomialQueue)(qPtr)
	if bootstrappedQ.IsEmpty() {
		// TODO I'll need to add some sort of check out thing or something whenever items are free from the list
		if q.freeQueueList.LengthGreaterThan(0) {
			fmt.Printf("RECURSION WORKS\n")
			return q.Dequeue()
		}
		fmt.Printf("but current count of thread shit: %d\n", q.freeQueueList.Count())
		fmt.Printf("empty return...\n")
		return nil, q
	}
	queuePriority, remainingBootstrappedQ := bootstrappedQ.DequeueWithCallback(
		q.dequeueCallback,
	)
	go q.freeQueueList.InsertObject(
		unsafe.Pointer(&remainingBootstrappedQ),
		qLessThanOther,
	)
	return queuePriority, q
}

func (q LazyMergeSkewBinomialQueue) dequeueCallback(childNodes []Node, remainingQueues ...SkewBinomialQueue) SkewBinomialQueue {
	go q.asyncDequeueCallback(childNodes, remainingQueues[1:]...)
	return remainingQueues[0]
}

func (q LazyMergeSkewBinomialQueue) transformAndInsert(skewQ SkewBinomialQueue) {
	bootstrappedQ := skewQToBootstrappedQ(skewQ)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&bootstrappedQ),
		qLessThanOther,
	)
}

func (q LazyMergeSkewBinomialQueue) asyncDequeueCallback(childNodes []Node, remainingQueues ...SkewBinomialQueue) {
	for _, skewQ := range remainingQueues {
		go q.transformAndInsert(skewQ)
	}

	var prioritiesRankZero []QueuePriority
	for _, child := range childNodes {
		if child.Rank() > 0 {
			validQ := newSkewBinomialQueue(
				child,
				nil,
			)
			go q.transformAndInsert(validQ)
		} else {
			prioritiesRankZero = append(
				prioritiesRankZero,
				child.Peek(),
			)
		}
	}
	lastQ := NewEmptySkewBinomialQueue().bulkInsert(prioritiesRankZero...)
	go q.transformAndInsert(lastQ)
	go q.meldFreeQueues()
}
