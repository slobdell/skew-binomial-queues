package skewBinomialQ

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var mutex = &sync.Mutex{}

/*
mutex.Lock()
defer mutex.Unlock()
*/

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
	//fmt.Printf("INSERTING OBJECTS\n")
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
			if t.listInBadState() {
				panic(fmt.Sprintf("%d: CASE 1 List is in bad state, value of Operation succeeded: %s", getGID(), 1))
			}
			t.InsertObject(object, lessThanFn)
		}
		return
	}

	// var previousForRetry *listNode = nil
	cursor := t.head
	debugCount := 0

	for {
		// SBL infinite loop here
		debugCount++
		//t.prettyPrint()
		// extra check needed for thread safety
		if cursor == nil {
			// SBL infinite loop here
			// THIS MEANS THAT THE QUEUE IS OUT OF ORDER, BUT WHERE I AM SUPPOSED TO PLACE IS MARKED
			/*
				asQ := *((*BootstrappedSkewBinomialQueue)(object))
				if debugCount == 14 {
					fmt.Printf("\n\n%s\n\n\n", asQ)
					t.prettyPrint()
				}
				fmt.Printf("INSERT CASE 1\n")
			*/
			fmt.Printf("%d\n", debugCount)
			// SBL try to isolate after which operation we're leaving shit in a bad state
			// SBL BUG HERE
			if t.listInBadState() {
				panic(fmt.Sprintf("%d: CASE 1 List is in bad state, value of Operation succeeded: %s", getGID(), 1))
			}
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
				cursor = cursor.markableNext.next
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
				t.InsertObject(object, lessThanFn)

				// Ideally we'd try again from the current node, but this will fail tests

				// try again from the current node
				// cursor = previousForRetry
			}
			break
		}
		// SBL CURRENTLY THIS CAN LOOP INFINITELY
		// previousForRetry = cursor
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

func (t *ThreadSafeList) deleteNode(previous, node *listNode) (success bool) {
	goID := getGID()
	if previous == nil {
		panic("case 0 how the fuck is it nil here")
	}
	nextNode := node.markableNext.next
	var previousMarkableNext *markablePointer
	currentNext := node.markableNext

	if currentNext.marked {
		//fmt.Printf("Fail case 1\n")
		return false
	}
	if previous != nil {
		previousMarkableNext = previous.markableNext
		if previousMarkableNext.marked {
			//fmt.Printf("Fail case 2\n")
			return false
		} else if previousMarkableNext.next != node {
			//fmt.Printf("Fail case 3\n")
			return false
		}
	}
	var newNext *markablePointer
	newNext = new(markablePointer)
	*newNext = markablePointer{
		marked: true,
		next:   nextNode,
	}
	fmt.Printf("%d Starting soft delete of %s\n", goID, node)
	operationSucceeded := atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&(node.markableNext))),
		unsafe.Pointer(currentNext),
		unsafe.Pointer(newNext),
	)
	fmt.Printf("%d End soft delete of %s, op succeeded: %s\n", goID, node, operationSucceeded)
	if !operationSucceeded {
		// SBL IT LOOKS AS THOUGH THE PROBLEM HAPPENS WHEN WE TRY AND DELETE 2 SIMULTANEOUSLY
		fmt.Printf("%d op did not succeed %s\n", goID, node)
		// SBL THIS APPEARS TO BE THE SOURCE OF THE PROBLEM...it's always NOT SUCCEEDING HERE
		// something was inserted while trying to delete
		return false
	}
	fmt.Printf("%d op succeeded\n", goID)
	// SBL IT APPEARS POSSIBLE FOR A GOROUTINE TO GET STUCK AT THIS POINT AND NOT CONTINUE!!!
	newNext = new(markablePointer)
	*newNext = markablePointer{
		marked: false,
		next:   nextNode,
	}
	if previous != nil {
		operationSucceeded = atomic.CompareAndSwapPointer(
			(*unsafe.Pointer)(unsafe.Pointer(&(previous.markableNext))),
			unsafe.Pointer(previousMarkableNext),
			unsafe.Pointer(newNext),
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
	fmt.Printf("%d Finished last compare and swap\n", goID)
	if !operationSucceeded {
		fmt.Printf("%d DELETE FAILED, ROLLING BACK\n", goID)
		fmt.Printf("%d Previous marked: %s\n", goID, previous.markableNext.marked)
		fmt.Printf("%d BEFORE Node marked: %s\n", goID, node.markableNext.marked)
		t.undeleteNode(node)
		fmt.Printf("%d AFTER Node marked: %s\n", goID, node.markableNext.marked)
		fmt.Printf("%d ROLLBACK COMPLETE\n\n", goID)
	}
	if operationSucceeded {
		fmt.Printf("%d last compare and swap succeed\n", goID)
	}
	fmt.Printf("%d Finished delete\n", goID)
	return operationSucceeded
}

func (t *ThreadSafeList) undeleteNode(node *listNode) {
	//fmt.Printf("spinning case 8\n")
	currentMarkableNext := node.markableNext
	if !currentMarkableNext.marked {
		return
	}
	newNext := &markablePointer{
		next: currentMarkableNext.next,
	}
	operationSucceeded := atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(&(node.markableNext))),
		unsafe.Pointer(currentMarkableNext),
		unsafe.Pointer(newNext),
	)
	if !operationSucceeded {
		t.undeleteNode(node)
	}
}

func (t *ThreadSafeList) hasMarks() bool {
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		time.Sleep(0)
		if cursor.markableNext.marked {
			return true
		}
	}
	return false
}
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func (t *ThreadSafeList) prettyPrint() {
	mutex.Lock()
	defer mutex.Unlock()
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		//asQ := *((*BootstrappedSkewBinomialQueue)(cursor.object))
		//fmt.Printf("N(%s)-", asQ)
		fmt.Printf("N-")
		if cursor.markableNext.marked {
			fmt.Printf("X")
		} else {
			fmt.Printf("-")
		}
		fmt.Printf("-->")
	}
	fmt.Printf("\n")
}

func (t *ThreadSafeList) doNothing() {
	/* debug code */
	time.Sleep(0)
}

func (t *ThreadSafeList) listInBadState() bool {
	var markedNodes []*listNode
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		if cursor.markableNext.marked {
			markedNodes = append(markedNodes, cursor)
		}
		t.doNothing()
	}
	if len(markedNodes) < 2 {
		return false
	}
	t.doNothing()
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		if cursor.markableNext.marked {
			for _, alreadyMarked := range markedNodes {
				if alreadyMarked == cursor {
					fmt.Printf("%d problem is with %s\n", getGID(), cursor)
					return true
				}
			}
		}
		t.doNothing()
	}
	return false
}

func (t *ThreadSafeList) purgeDeleted() {
	/* Simple fallback strategy for failed deletes.  After a soft-delete, if the 2nd
	operation doesn't succeed, we can just scan the list and update references. */
	panic("DO NOT USE THIS")
	var previous *listNode
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {

		if cursor.markableNext.marked {
			if previous != nil {

				previousMarkableNext := previous.markableNext
				newNext := &markablePointer{
					next: cursor.markableNext.next,
				}
				atomic.CompareAndSwapPointer(
					(*unsafe.Pointer)(unsafe.Pointer(&(previous.markableNext))),
					unsafe.Pointer(previousMarkableNext),
					unsafe.Pointer(newNext),
				)
			} else {
				panic("TODO handle case of deleting head")
			}
			//fmt.Printf("spin case 1\n")
		}

		previous = cursor
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
		//fmt.Printf("sping case 8\n")
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
		currentNode = t.head
		if currentNode == nil {
			return returnObjects
		} else if currentNode.markableNext.marked {
			return returnObjects
		}
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
		if params.previousNode != nil && params.currentNode != nil {
			if params.previousNode.markableNext.next != params.currentNode {
				//fmt.Printf("I expected failure\n")
			}
		}
		panic("DONE WITH THIS RIGHT?")
		operationSucceeded := t.deleteNode(params.previousNode, currentNode)
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
		time.Sleep(0)
	}
	return count
}

func (t ThreadSafeList) PopNth(n int, failAddress unsafe.Pointer) unsafe.Pointer {
	count := 0
	var previous *listNode
	for cursor := t.head; cursor != nil; cursor = cursor.markableNext.next {
		count++
		if count >= n {
			//fmt.Printf("sping case 6\n")
			operationSucceeded := t.deleteNode(previous, cursor)
			if operationSucceeded {
				return cursor.object
			} else {
				time.Sleep(0)
			}
		}
		previous = cursor
		/*
			if count > n*2 {
				// TODO probably a better way to do this
				break
			}
		*/
	}
	//fmt.Printf("RETURNING FAIL\n")
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

func (t *ThreadSafeList) PopFirst() unsafe.Pointer {
	// TODO add tests
	currentHead := t.head
	if currentHead == nil {
		return nil
	}
	operationSucceeded := t.deleteNode(nil, currentHead)
	if operationSucceeded {
		return currentHead.object
	}
	return t.PopFirst()
}

func (t ThreadSafeList) Repr() {
	// TODO delete this function
	var objects []unsafe.Pointer
	for item := range t.Iter() {
		objects = append(objects, item)
	}
	for outer_idx, obj1 := range objects {
		for inner_idx, obj2 := range objects {
			if inner_idx == outer_idx {
				continue
			}
			if obj1 == obj2 {
				panic("duplicate objects exist in the list")
			}
			//fmt.Printf("sping case 4\n")
		}
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
			//fmt.Printf("sping case 3\n")
		}
		close(ch)
	}()
	return ch
}

func NewThreadSafeList() ThreadSafeList {
	return ThreadSafeList{}
}
