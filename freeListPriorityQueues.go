package skewBinomialQ

import (
	"time"
	"unsafe"
)

const MIN_QUEUE_SIZE = 2

func qLessThanOther(q1 unsafe.Pointer, q2 unsafe.Pointer) bool {
	priorityQ1 := (*SkewBinomialQueue)(q1)
	priorityQ2 := (*SkewBinomialQueue)(q2)
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
	freeQueueList *ThreadSafeList
}

func NewEmptyLazyMergeSkewBinomialQueue() PriorityQueue {
	primitiveQ := NewEmptySkewBinomialQueue()
	threadSafeList := ThreadSafeList{}
	threadSafeList.InsertObject(
		unsafe.Pointer(&primitiveQ),
		qLessThanOther,
	)

	return LazyMergeSkewBinomialQueue{
		freeQueueList: &threadSafeList,
	}
}

func (q LazyMergeSkewBinomialQueue) Enqueue(priority QueuePriority) PriorityQueue {
	sizeOneQ := NewEmptySkewBinomialQueue().Enqueue(priority).(SkewBinomialQueue)
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
	var queues []unsafe.Pointer = q.freeQueueList.TrySplitLastN(
		TrySplitParameter{
			N: 2,
		},
	)
	if len(queues) < 2 {
		for _, queuePtr := range queues {
			q.freeQueueList.InsertObject(
				queuePtr,
				qLessThanOther,
			)
		}
	} else {
		q1 := *((*SkewBinomialQueue)(queues[0]))
		q2 := *((*SkewBinomialQueue)(queues[1]))
		finalQ := (q1.Meld(q2)).(SkewBinomialQueue)
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
	panic("TEMP")
	var qPtr unsafe.Pointer
	for {
		qPtr = q.freeQueueList.Peek()
		if qPtr == nil {
			// explicitly yield to another goroutine
			time.Sleep(0)
		}
		break
	}
	firstQ := (*SkewBinomialQueue)(qPtr)
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
	length := 0
	for qPtr := range q.freeQueueList.Iter() {
		priorityQ := (*SkewBinomialQueue)(qPtr)
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
	panic("TEMP")
	/*
		This is the only funciton where coupling occurs between this struct and the child structure
	*/
	// TODO so I think PopFirst() will only return nil if there are concurrent operations. Need to verify that this will not create an infinite loop or whatever...
	var qPtr unsafe.Pointer
	for {
		qPtr = q.freeQueueList.PopFirst()
		if qPtr == nil {
			// explicitly yield to another goroutine
			time.Sleep(0)
		}
		break
	}
	primitiveQ := (*SkewBinomialQueue)(qPtr)
	if primitiveQ.IsEmpty() {
		return nil, q
	}
	poppedQueue, remainingQueue := primitiveQ.popHighestPriorityQueue()
	returnQueuePriority := poppedQueue.Peek()
	go q.freeChildren(poppedQueue.heapHead.Children(), remainingQueue)
	return returnQueuePriority, q
}

func (q LazyMergeSkewBinomialQueue) freeChildren(childNodes []Node, remainingQueue SkewBinomialQueue) {
	panic("TEMP")
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&remainingQueue),
		qLessThanOther,
	)

	var prioritiesRankZero []QueuePriority
	for _, child := range childNodes {
		if child.Rank() > 0 {
			validQ := newSkewBinomialQueue(
				child,
				nil,
			)
			q.freeQueueList.InsertObject(
				unsafe.Pointer(&validQ),
				qLessThanOther,
			)
		} else {
			prioritiesRankZero = append(
				prioritiesRankZero,
				child.Peek(),
			)
		}
	}
	lastQ := NewEmptySkewBinomialQueue().bulkInsert(prioritiesRankZero...)
	q.freeQueueList.InsertObject(
		unsafe.Pointer(&lastQ),
		qLessThanOther,
	)
	go q.meldFreeQueues()
}
