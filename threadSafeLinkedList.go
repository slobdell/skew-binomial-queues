package skewBinomialQ

import (
	"sync/atomic"
	"unsafe"
)

type SpinLock struct {
	state int32
}

func (s *SpinLock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&s.state, 0, 1)
}

func (s SpinLock) IsLocked() bool {
	return s.state == 1
}

func (s *SpinLock) Unlock() {
	s.state = 0
}

type listNode struct {
	markableNext *markablePointer
	object       unsafe.Pointer
	spinLock     SpinLock
}

type ThreadSafeList struct {
	head     *listNode
	spinLock SpinLock
}

type TrySplitParameter struct {
	N            int
	currentNode  *listNode
	previousNode *listNode
}

type markablePointer struct {
	marked bool
	next   *listNode
}

func (t *ThreadSafeList) InsertObject(object unsafe.Pointer, lessThanFn func(unsafe.Pointer, unsafe.Pointer) bool) {
	currentHeadAddress := &t.head
	currentHead := t.head

	// go doesn't use short circuiting so we need to do this sort of whacky logic
	shouldInsertAtHead := currentHead == nil
	if !shouldInsertAtHead {
		shouldInsertAtHead = lessThanFn(object, currentHead.object)
	}
	if shouldInsertAtHead {
		newNode := listNode{
			object: object,
			markableNext: &markablePointer{
				next: currentHead,
			},
		}

		operationSucceeded := atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(currentHeadAddress)),
			unsafe.Pointer(currentHead),
			unsafe.Pointer(&newNode),
		)
		if !operationSucceeded {
			// retry here is cheap since it's at head
			t.InsertObject(object, lessThanFn)
		}
		return
	}

	var previousForRetry *listNode = nil
	cursor := t.head
	for {
		// extra check needed for thread safety
		if cursor == nil {
			t.InsertObject(object, lessThanFn)
			return
		}
		nextNode := cursor.markableNext.next
		// get around no short circuiting
		shouldInsertNext := nextNode == nil
		if !shouldInsertNext {
			shouldInsertNext = lessThanFn(object, nextNode.object)
		}
		if shouldInsertNext {
			currentNext := cursor.markableNext
			if currentNext.marked {
				continue
			}
			newNode := listNode{
				object: object,
				markableNext: &markablePointer{
					next: currentNext.next,
				},
			}
			newNext := markablePointer{
				next: &newNode,
			}
			operationSucceeded := atomic.CompareAndSwapPointer(
				(*unsafe.Pointer)(unsafe.Pointer(&(cursor.markableNext))),
				unsafe.Pointer(currentNext),
				unsafe.Pointer(&newNext),
			)
			if operationSucceeded {
				break
			} else {
				// try again from the current node
				cursor = previousForRetry
			}
			break
		}
		previousForRetry = cursor
		cursor = cursor.markableNext.next
	}
}

func (t *ThreadSafeList) PopHead() unsafe.Pointer {
	currentHeadAddress := &t.head
	currentHead := t.head
	if t.head == nil {
		return nil
	}
	newHead := currentHead.markableNext.next
	operationSucceeded := atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(currentHeadAddress)),
		unsafe.Pointer(currentHead),
		unsafe.Pointer(newHead),
	)
	if !operationSucceeded {
		return t.PopHead()
	}
	return currentHead.object
}

func (t *ThreadSafeList) deleteNode(node, previous *listNode) (success bool) {
	nextNode := node.markableNext.next
	newNext := markablePointer{
		marked: true,
		next:   nextNode,
	}
	operationSucceeded := atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&(node.markableNext))),
		unsafe.Pointer(node.markableNext),
		unsafe.Pointer(&newNext),
	)
	if !operationSucceeded {
		// something was inserted while trying to delete
		return false
	}
	newNext = markablePointer{
		next: nextNode,
	}
	if previous != nil {
		operationSucceeded = atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&(previous.markableNext))),
			unsafe.Pointer(previous.markableNext),
			unsafe.Pointer(&newNext),
		)
	} else {
		// we just delete head
		currentHeadAddress := &t.head
		currentHead := t.head
		operationSucceeded = atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(currentHeadAddress)),
			unsafe.Pointer(currentHead),
			unsafe.Pointer(nextNode),
		)
	}
	return operationSucceeded
}

func (t *ThreadSafeList) DeleteObject(object unsafe.Pointer) (success bool) {
	var previous *listNode
	cursor := t.head
	for {
		if cursor == nil {
			break
		}
		if cursor.object == object {
			operationSucceeded := t.deleteNode(cursor, previous)
			if !operationSucceeded {
				return t.DeleteObject(object)
			}
			return true
		}

		previous = cursor
		cursor = cursor.markableNext.next
	}
	return false
}

func (t *ThreadSafeList) TrySplitLastN(params TrySplitParameter) []unsafe.Pointer {
	/*
		In the multi-threaded environment, it's possible during this split function
		to split greater than or fewer objects specified by the function parameters
		if concurrent adds or deletes are happening.
	*/
	var returnObjects []unsafe.Pointer
	currentNode := params.currentNode
	currentHead := t.head
	if currentHead == nil || params.N == 0 {
		return returnObjects
	}
	if currentNode == nil {
		currentNode = currentHead
	}
	var downstreamReturnObjects []unsafe.Pointer
	nextNode := currentNode.markableNext.next
	if nextNode != nil {
		downstreamReturnObjects = t.TrySplitLastN(
			TrySplitParameter{
				N:            params.N,
				currentNode:  nextNode,
				previousNode: currentNode,
			},
		)
	}
	if len(downstreamReturnObjects) < params.N {
		objToPop := currentNode.object
		operationSucceeded := t.deleteNode(currentNode, params.previousNode)
		if operationSucceeded {
			returnObjects = append(downstreamReturnObjects, objToPop)
		} else {
			returnObjects = downstreamReturnObjects
		}
		return returnObjects
	}
	return downstreamReturnObjects
}

func (t ThreadSafeList) Count() int {
	count := 0
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		count++
	}
	return count
}

func (t ThreadSafeList) LengthGreaterThan(n int) bool {
	// TODO add tests
	count := 0
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		count++
		if count > n {
			return true
		}
	}
	return false
}

func (t ThreadSafeList) Peek() unsafe.Pointer {
	if t.head == nil {
		return nil
	}
	return t.head.object
}

func (t *ThreadSafeList) PopFirst() unsafe.Pointer {
	// TODO add tests
	currentHead := t.head
	if currentHead == nil {
		return nil
	}
	operationSucceeded := t.deleteNode(currentHead, nil)
	if operationSucceeded {
		return currentHead.object
	}
	return t.PopFirst()
}

func (t ThreadSafeList) Iter() <-chan unsafe.Pointer {
	ch := make(chan unsafe.Pointer)
	go func() {
		cursor := t.head
		for {
			if cursor == nil {
				break
			}
			ch <- cursor.object
			cursor = cursor.markableNext.next
		}
		close(ch)
	}()
	return ch
}

func NewThreadSafeList() ThreadSafeList {
	return ThreadSafeList{}
}
