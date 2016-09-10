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

type markablePointer struct {
	marked bool
	next   *listNode
}

func (t *ThreadSafeList) InsertObject(object unsafe.Pointer, lessThanFn func(unsafe.Pointer, unsafe.Pointer) bool) {
	currentHeadAddress := &t.head
	currentHead := t.head

	if currentHead == nil || lessThanFn(object, currentHead.object) {
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
			t.InsertObject(object, lessThanFn)
		}
		return
	}

	cursor := t.head
	for {
		if cursor.markableNext.next == nil || lessThanFn(object, cursor.markableNext.next.object) {
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
			if !operationSucceeded {
				t.InsertObject(object, lessThanFn)
				return
			}
			break
		}
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

func (t *ThreadSafeList) DeleteObject(object unsafe.Pointer) {
	var previous *listNode
	currentHeadAddress := &t.head
	currentHead := t.head
	cursor := currentHead
	for {
		if cursor == nil {
			break
		}
		if cursor.object == object {
			nextNode := cursor.markableNext.next
			newNext := markablePointer{
				marked: true,
				next:   nextNode,
			}
			operationSucceeded := atomic.CompareAndSwapPointer(
				(*unsafe.Pointer)(unsafe.Pointer(&(cursor.markableNext))),
				unsafe.Pointer(cursor.markableNext),
				unsafe.Pointer(&newNext),
			)
			if !operationSucceeded {
				t.DeleteObject(object)
				return
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
				// we just deleted head
				operationSucceeded = atomic.CompareAndSwapPointer(
					(*unsafe.Pointer)(unsafe.Pointer(currentHeadAddress)),
					unsafe.Pointer(currentHead),
					unsafe.Pointer(nextNode),
				)
			}
			if !operationSucceeded {
				t.DeleteObject(object)
			}
			break
		}

		previous = cursor
		cursor = cursor.markableNext.next
	}
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
