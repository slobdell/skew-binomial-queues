package skewBinomialQ

import (
	"sync/atomic"
	"time"
	"unsafe"
)

type listNode struct {
	object       unsafe.Pointer
	markableNext *markablePointer
}

type ThreadSafeList struct {
	head *listNode
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
				marked: false,
				next:   currentHead,
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

	var previous *listNode = nil
	var operationSucceeded bool = false
	for cursor := t.head; !operationSucceeded; cursor = cursor.markableNext.next {
		// extra check needed for thread safety
		if cursor == nil {
			// SBL BUG HERE
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
				t.hardDeleteNode(previous, cursor)
				continue
			}
			newNext := markablePointer{
				next: &listNode{
					object: object,
					markableNext: &markablePointer{
						next: currentNext.next,
					},
				},
			}
			operationSucceeded = atomic.CompareAndSwapPointer(
				(*unsafe.Pointer)(unsafe.Pointer(&(cursor.markableNext))),
				unsafe.Pointer(currentNext),
				unsafe.Pointer(&newNext),
			)
			if !operationSucceeded {
				// FIXME Ideally we'd try again from the current node, but this will fail tests
				// For now go with the simplest approach
				t.InsertObject(object, lessThanFn)
				return
			}
		}
		previous = cursor
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

func (t *ThreadSafeList) deleteNode(previous, node *listNode) (success bool) {
	currentNext := node.markableNext

	if currentNext.marked {
		return false
	}

	if previous != nil {
		previousMarkableNext := previous.markableNext
		if previousMarkableNext.marked {
			return false
		} else if previousMarkableNext.next != node {
			return false
		}
	}

	newNext := &markablePointer{
		marked: true,
		next:   node.markableNext.next,
	}

	operationSucceeded := atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&(node.markableNext))),
		unsafe.Pointer(currentNext),
		unsafe.Pointer(newNext),
	)
	if !operationSucceeded {
		return false
	}

	// at this point, rather than deal with the possibility of possibly unrolling after a failed
	// compare and swap, we will instead have any traversal of a deleted node update the pointer
	// reference.  Otherwise, it's possible for the operations above to create a condition
	// in which no other goroutines make progress, and the current goroutine ends up getting
	// starved out, and the 2nd compare and swap will never happen
	t.hardDeleteNode(previous, node)
	return true
}

func (t *ThreadSafeList) hardDeleteNode(previous, node *listNode) {
	nextNode := node.markableNext.next
	if !node.markableNext.marked {
		return
	}
	newNext := new(markablePointer)
	*newNext = markablePointer{
		marked: false,
		next:   nextNode,
	}
	if previous != nil {
		var previousMarkableNext *markablePointer = previous.markableNext
		if previousMarkableNext.next != node {
			return
		}
		if previousMarkableNext.marked {
			return
		}
		atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&(previous.markableNext))),
			unsafe.Pointer(previousMarkableNext),
			unsafe.Pointer(newNext),
		)
	} else {
		// we just delete head
		atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&(t.head))),
			unsafe.Pointer(node),
			unsafe.Pointer(nextNode),
		)
	}
}

func (t *ThreadSafeList) DeleteObject(object unsafe.Pointer) (success bool) {
	var previous *listNode
	cursor := t.head
	for {
		if cursor == nil {
			break
		}
		if cursor.object == object {
			operationSucceeded := t.deleteNode(previous, cursor)
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

func (t ThreadSafeList) Count() int {
	count := 0
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		count++
		time.Sleep(0)
	}
	return count
}

func (t ThreadSafeList) PopNth(n int, failAddress unsafe.Pointer) unsafe.Pointer {
	count := 0
	var previous *listNode
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		if cursor.markableNext.marked {
			t.hardDeleteNode(previous, cursor)
		}
		count++
		if count >= n {
			operationSucceeded := t.deleteNode(previous, cursor)
			if operationSucceeded {
				return cursor.object
			} else {
				time.Sleep(0)
			}
		}
		previous = cursor
	}
	return failAddress
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

func (t *ThreadSafeList) PopFirst(failAddress unsafe.Pointer) unsafe.Pointer {
	// TODO explore performance benefits of not using recursion
	currentHead := t.head
	if currentHead == nil {
		return failAddress
	} else if currentHead.markableNext.marked {
		t.hardDeleteNode(nil, currentHead)
		return t.PopFirst(failAddress)
	}
	operationSucceeded := t.deleteNode(nil, currentHead)
	if operationSucceeded {
		return currentHead.object
	}
	return t.PopFirst(failAddress)
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
