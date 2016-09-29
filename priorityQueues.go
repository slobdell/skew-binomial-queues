package skewBinomialQ

import (
	"fmt"
	"sort"
)

type QueuePriority interface {
	LessThan(otherPriority QueuePriority) bool
}

func min(x, y QueuePriority) QueuePriority {
	if x.LessThan(y) {
		return x
	}
	return y
}

type Node interface {
	Rank() int
	Length() int
	Children() []Node
	Peek() QueuePriority
	Link(heaps ...Node) Node
	IsEmpty() bool
}

type PriorityQueue interface {
	Enqueue(priority QueuePriority) PriorityQueue
	Peek() QueuePriority
	Meld(otherQ PriorityQueue) PriorityQueue
	Dequeue() (QueuePriority, PriorityQueue)
	Length() int
	IsEmpty() bool
}

var NULL_OBJECT = NullObject{}

type NullObject struct {
	head     *NullObject
	children []Node
}

func (n NullObject) Link(heaps ...Node) Node {
	return n
}

func (n NullObject) Peek() QueuePriority {
	return nil
}

func (n NullObject) Children() []Node {
	return n.children
}

func (n NullObject) Rank() int {
	return -1
}

func (n NullObject) IsEmpty() bool {
	return true
}

func (n NullObject) Length() int {
	return 0
}

type SkewBinomialHeap struct {
	rank     int
	priority QueuePriority
	children []Node // this is possibly problematic but I'm not sure..pointer vs non pointer
	length   int
}

type skewByRank []SkewBinomialQueue

func (a skewByRank) Len() int           { return len(a) }
func (a skewByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a skewByRank) Less(i, j int) bool { return a[i].rank() > a[j].rank() }

type byRank []Node

func (a byRank) Len() int           { return len(a) }
func (a byRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRank) Less(i, j int) bool { return a[i].Rank() < a[j].Rank() }

type byPriority []Node

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Peek().LessThan(a[j].Peek()) }

func NewSkewBinomialHeap(priority QueuePriority) Node {
	return SkewBinomialHeap{
		priority: priority,
		length:   1,
	}
}

func highestPriorityHeap(heaps ...Node) Node {
	sort.Sort(
		byPriority(
			heaps,
		),
	)
	return heaps[0]
}

func lowerRankHeaps(heaps ...Node) []Node {
	sort.Sort(
		byPriority(
			heaps,
		),
	)
	heaps = heaps[1:]
	sort.Sort(
		byRank(
			heaps,
		),
	)
	return heaps
}

func newNode(priority QueuePriority, rank int, children []Node) Node {
	var existingLength int
	for _, node := range children {
		// TODO see if I can do this more elegantly?
		existingLength += node.Length()
	}
	return SkewBinomialHeap{
		rank:     rank,
		priority: priority,
		children: children,
		length:   1 + existingLength,
	}
}

func (s SkewBinomialHeap) IsEmpty() bool {
	return false
}

func (s SkewBinomialHeap) Children() []Node {
	return s.children
}

func (s SkewBinomialHeap) Link(heaps ...Node) Node {
	heaps = append(heaps, s)
	sort.Sort(
		byRank(
			heaps,
		),
	)
	newRank := heaps[len(heaps)-1].Rank() + 1
	highestPriorityHeap := highestPriorityHeap(heaps...)
	return newNode(
		highestPriorityHeap.Peek(),
		newRank,
		append(
			lowerRankHeaps(heaps...),
			highestPriorityHeap.Children()...,
		),
	)
}

func (s SkewBinomialHeap) Length() int {
	return s.length
}

func (s SkewBinomialHeap) Rank() int {
	return s.rank
}

func (s SkewBinomialHeap) Peek() QueuePriority {
	return s.priority
}

type SkewBinomialQueue struct {
	heapHead     Node
	rightSibling *SkewBinomialQueue
}

func NewEmptySkewBinomialQueue() SkewBinomialQueue {
	return SkewBinomialQueue{
		heapHead:     NULL_OBJECT,
		rightSibling: nil,
	}
}

func newSkewBinomialQueue(heapHead Node, rightSibling *SkewBinomialQueue) SkewBinomialQueue {
	return SkewBinomialQueue{
		heapHead:     heapHead,
		rightSibling: rightSibling,
	}
}

func (q SkewBinomialQueue) String() string {
	return fmt.Sprintf("<Skew Binomial Queue, peek value: %s, Length: %d>", q.Peek(), q.Length())
}

func (q SkewBinomialQueue) firstTwoTreesEqualRank() bool {
	if q.heapHead.IsEmpty() {
		return false
	}
	if q.rightSibling == nil {
		return false
	}
	return q.rank() == q.rightSibling.rank()
}

func (q SkewBinomialQueue) Enqueue(priority QueuePriority) PriorityQueue {
	if q.firstTwoTreesEqualRank() {
		// null checks should not be necessary here since first two trees are equal
		return newSkewBinomialQueue(
			NewSkewBinomialHeap(priority).Link(
				q.heapHead,
				q.rightSibling.heapHead,
			),
			q.rightSibling.rightSibling,
		)
	}
	return newSkewBinomialQueue(
		NewSkewBinomialHeap(priority),
		&q,
	)
}

func (q SkewBinomialQueue) Peek() QueuePriority {
	if q.heapHead.IsEmpty() {
		return nil
	}
	if q.rightSibling == nil {
		return q.heapHead.Peek()
	}
	if q.rightSibling.heapHead.IsEmpty() {
		return q.heapHead.Peek()
	}
	return min(
		q.heapHead.Peek(),
		q.rightSibling.Peek(),
	)
}

func (q SkewBinomialQueue) Meld(otherQ PriorityQueue) PriorityQueue {
	if q.IsEmpty() {
		return otherQ
	} else if otherQ.IsEmpty() {
		return q
	}

	otherSkewQ, ok := otherQ.(SkewBinomialQueue)
	if !ok {
		panic("Meld unequal priority queue types")
	}

	allForests := append(q.asIndividualForests(), otherSkewQ.asIndividualForests()...)
	sort.Sort(
		skewByRank(
			allForests,
		),
	)
	finalQ := new(SkewBinomialQueue)

	*finalQ = NewEmptySkewBinomialQueue()

	for _, qOneTree := range allForests {
		if finalQ.heapHead.IsEmpty() {
			// TODO avoid special casing this if I can
			*finalQ = newSkewBinomialQueue(
				qOneTree.heapHead,
				nil,
			)
		} else if finalQ.rank() == qOneTree.rank() {
			if finalQ.rightSibling == nil {
				*finalQ = newSkewBinomialQueue(
					finalQ.heapHead.Link(qOneTree.heapHead),
					nil,
				)
			} else {
				priorityQ := finalQ.rightSibling.Meld(
					newSkewBinomialQueue(
						finalQ.heapHead.Link(qOneTree.heapHead),
						nil,
					),
				)
				*finalQ, ok = priorityQ.(SkewBinomialQueue)
			}
		} else if qOneTree.rank() < finalQ.rank() {
			newQ := newSkewBinomialQueue(
				qOneTree.heapHead,
				finalQ,
			)
			finalQ = new(SkewBinomialQueue)
			*finalQ = newQ
		} else {
			*finalQ = newSkewBinomialQueue(
				finalQ.heapHead,
				&qOneTree,
			)
		}
	}
	return *finalQ
}

func (q SkewBinomialQueue) rank() int {
	// TODO unsure if this method makes this unclear, but most use cases for checking
	// rank involve cheaking the heap head
	return q.heapHead.Rank()
}

func (q SkewBinomialQueue) asIndividualForests() []SkewBinomialQueue {
	if q.IsEmpty() {
		return nil
	} else if q.rightSibling == nil {
		return []SkewBinomialQueue{q}
	}
	return append(
		[]SkewBinomialQueue{newSkewBinomialQueue(q.heapHead, nil)},
		q.rightSibling.asIndividualForests()...,
	)
}

func (q SkewBinomialQueue) Dequeue() (QueuePriority, PriorityQueue) {
	return q.DequeueWithMergeCallback(
		q.dequeueMergeCallback,
	)
}

func (q SkewBinomialQueue) DequeueWithMergeCallback(callback func([]Node, ...*SkewBinomialQueue) SkewBinomialQueue) (QueuePriority, PriorityQueue) {
	if q.IsEmpty() {
		return nil, q
	}
	poppedQueue, remainingQueue := q.popHighestPriorityQueue()
	return poppedQueue.Peek(), callback(poppedQueue.heapHead.Children(), &remainingQueue)
}

func (q SkewBinomialQueue) dequeueMergeCallback(childNodes []Node, remainingQueues ...*SkewBinomialQueue) SkewBinomialQueue {
	var childrenRankGreaterThanZero []SkewBinomialQueue
	var prioritiesRankZero []QueuePriority
	remainingQPtr := remainingQueues[0]
	remainingQueue := *remainingQPtr
	for _, skewQ := range remainingQueues[1:] {
		newlyAllocatedItem := skewQ
		remainingQueue = remainingQueue.Meld(*newlyAllocatedItem).(SkewBinomialQueue)
	}
	for _, child := range childNodes {
		if child.Rank() > 0 {
			childrenRankGreaterThanZero = append(
				childrenRankGreaterThanZero,
				newSkewBinomialQueue(
					child,
					nil,
				),
			)
		} else {
			prioritiesRankZero = append(
				prioritiesRankZero,
				child.Peek(),
			)
		}
	}

	finalQueue := remainingQueue
	for _, queue := range childrenRankGreaterThanZero {
		finalQueue, _ = finalQueue.Meld(queue).(SkewBinomialQueue)
	}
	return finalQueue.bulkInsert(prioritiesRankZero...)
}

func (q SkewBinomialQueue) Length() int {
	if q.rightSibling == nil || q.rightSibling.heapHead.IsEmpty() {
		return q.heapHead.Length()
	}
	return q.heapHead.Length() + q.rightSibling.Length()
}

func (q SkewBinomialQueue) bulkInsert(priorities ...QueuePriority) SkewBinomialQueue {
	if len(priorities) == 0 {
		return q
	}
	return q.Enqueue(
		priorities[0],
	).(SkewBinomialQueue).bulkInsert(priorities[1:]...)
}

func (q SkewBinomialQueue) IsEmpty() bool {
	return q.heapHead.IsEmpty()
}

func (q SkewBinomialQueue) popHighestPriorityQueue() (SkewBinomialQueue, SkewBinomialQueue) {
	if q.rightSibling == nil {
		return q, NewEmptySkewBinomialQueue()
	}
	if q.rightSibling.heapHead.IsEmpty() {
		return newSkewBinomialQueue(
				q.heapHead,
				nil,
			),
			*q.rightSibling
	}
	if q.heapHead.Peek().LessThan(q.rightSibling.Peek()) {
		return newSkewBinomialQueue(
				q.heapHead,
				nil,
			),
			*q.rightSibling
	}
	poppedQueue, remainingQueue := q.rightSibling.popHighestPriorityQueue()
	return poppedQueue, newSkewBinomialQueue(q.heapHead, &remainingQueue)
}

type RootedSkewBinomialQueue struct {
	highestPriorityObject QueuePriority
	priorityQueue         SkewBinomialQueue
}

func NewEmptyRootedSkewBinomialQueue() RootedSkewBinomialQueue {
	return RootedSkewBinomialQueue{
		highestPriorityObject: nil,
		priorityQueue:         NewEmptySkewBinomialQueue(),
	}
}

func (r RootedSkewBinomialQueue) IsEmpty() bool {
	return r.highestPriorityObject == nil
}

func (r RootedSkewBinomialQueue) Enqueue(priority QueuePriority) RootedSkewBinomialQueue {
	if r.IsEmpty() {
		return RootedSkewBinomialQueue{
			highestPriorityObject: priority,
			priorityQueue:         NewEmptySkewBinomialQueue(),
		}
	}
	if r.highestPriorityObject.LessThan(priority) {
		return RootedSkewBinomialQueue{
			highestPriorityObject: r.highestPriorityObject,
			priorityQueue:         r.priorityQueue.Enqueue(priority).(SkewBinomialQueue),
		}
	}
	return RootedSkewBinomialQueue{
		highestPriorityObject: priority,
		priorityQueue: r.priorityQueue.Enqueue(
			r.highestPriorityObject,
		).(SkewBinomialQueue),
	}
}

func (r RootedSkewBinomialQueue) Dequeue() (QueuePriority, RootedSkewBinomialQueue) {
	if r.priorityQueue.IsEmpty() {
		return r.highestPriorityObject, NewEmptyRootedSkewBinomialQueue()
	}
	highestPriorityObject, priorityQueue := r.priorityQueue.Dequeue()
	return r.highestPriorityObject, RootedSkewBinomialQueue{
		highestPriorityObject: highestPriorityObject,
		priorityQueue:         priorityQueue.(SkewBinomialQueue),
	}
}

func (r RootedSkewBinomialQueue) Meld(otherQ RootedSkewBinomialQueue) RootedSkewBinomialQueue {
	mergedPrimitiveQueue := r.priorityQueue.Meld(otherQ.priorityQueue)
	if r.highestPriorityObject.LessThan(otherQ.highestPriorityObject) {
		return RootedSkewBinomialQueue{
			highestPriorityObject: r.highestPriorityObject,
			priorityQueue: mergedPrimitiveQueue.Enqueue(
				otherQ.highestPriorityObject,
			).(SkewBinomialQueue),
		}
	}
	return RootedSkewBinomialQueue{
		highestPriorityObject: otherQ.highestPriorityObject,
		priorityQueue: mergedPrimitiveQueue.Enqueue(
			r.highestPriorityObject,
		).(SkewBinomialQueue),
	}
}

func (r RootedSkewBinomialQueue) Peek() QueuePriority {
	return r.highestPriorityObject
}

func (r RootedSkewBinomialQueue) Length() int {
	if r.IsEmpty() {
		return 0
	}
	return 1 + r.priorityQueue.Length()
}

type BootstrappedSkewBinomialQueue struct {
	highestPriorityObject QueuePriority
	priorityQueue         *SkewBinomialQueue
	length                int
}

func NewEmptyBootstrappedSkewBinomialQueue() BootstrappedSkewBinomialQueue {
	newEmptySkewQ := NewEmptySkewBinomialQueue()
	return BootstrappedSkewBinomialQueue{
		highestPriorityObject: nil,
		priorityQueue:         &newEmptySkewQ,
		length:                0,
	}
}

func (b BootstrappedSkewBinomialQueue) String() string {
	return fmt.Sprintf("<BootstrappedSkewBinomialQueue, peek value: %d, length: %d>", b.Peek(), b.Length())
}

func (b BootstrappedSkewBinomialQueue) IsEmpty() bool {
	return b.highestPriorityObject == nil
}

func (b BootstrappedSkewBinomialQueue) Enqueue(priority QueuePriority) BootstrappedSkewBinomialQueue {
	newEmptySkewQ := NewEmptySkewBinomialQueue()
	return b.Meld(
		BootstrappedSkewBinomialQueue{
			highestPriorityObject: priority,
			priorityQueue:         &newEmptySkewQ,
			length:                1,
		},
	)
}

func (b BootstrappedSkewBinomialQueue) Peek() QueuePriority {
	return b.highestPriorityObject
}

func (b BootstrappedSkewBinomialQueue) Dequeue() (QueuePriority, BootstrappedSkewBinomialQueue) {
	return b.DequeueWithMergeCallback(
		b.priorityQueue.dequeueMergeCallback,
	)
}

func (b BootstrappedSkewBinomialQueue) DequeueWithMergeCallback(mergeCallback func([]Node, ...*SkewBinomialQueue) SkewBinomialQueue) (QueuePriority, BootstrappedSkewBinomialQueue) {
	if b.priorityQueue.IsEmpty() || b.IsEmpty() {
		return b.highestPriorityObject, NewEmptyBootstrappedSkewBinomialQueue()
	}
	queuePriority, updatedPrimitiveQueue := b.priorityQueue.DequeueWithMergeCallback(mergeCallback)
	highestPriorityBootstrappedQueue, ok := queuePriority.(BootstrappedSkewBinomialQueue)

	if !ok {
		panic("Cannot cast to a Bootstrapped Queue. This case should not have been reached")
	}
	updatedSkewQ := updatedPrimitiveQueue.(SkewBinomialQueue)
	mergedPrimitiveQ := mergeCallback(
		[]Node{},
		&updatedSkewQ,
		highestPriorityBootstrappedQueue.priorityQueue,
	)

	updatedBootstrappedQueue := BootstrappedSkewBinomialQueue{
		highestPriorityObject: highestPriorityBootstrappedQueue.highestPriorityObject,
		priorityQueue:         &mergedPrimitiveQ,
		length:                b.Length() - 1, // FIXME this no longer works w callback, value is wrong
	}
	return b.highestPriorityObject, updatedBootstrappedQueue
}

func (b BootstrappedSkewBinomialQueue) Length() int {
	return b.length
}

func (b BootstrappedSkewBinomialQueue) Meld(otherQ BootstrappedSkewBinomialQueue) BootstrappedSkewBinomialQueue {
	if b.IsEmpty() {
		return otherQ
	} else if otherQ.IsEmpty() {
		return b
	} else if b.Peek().LessThan(otherQ.Peek()) {
		priorityQueue := b.priorityQueue.Enqueue(
			otherQ,
		).(SkewBinomialQueue)
		return BootstrappedSkewBinomialQueue{
			highestPriorityObject: b.Peek(),
			priorityQueue:         &priorityQueue,
			length:                b.Length() + otherQ.Length(),
		}
	}
	primitiveQueue := otherQ.priorityQueue.Enqueue(
		b,
	).(SkewBinomialQueue)

	return BootstrappedSkewBinomialQueue{
		highestPriorityObject: otherQ.Peek(),
		priorityQueue:         &primitiveQueue,
		length:                b.Length() + otherQ.Length(),
	}
}

func (b BootstrappedSkewBinomialQueue) LessThan(otherPriority QueuePriority) bool {
	otherQ, ok := otherPriority.(BootstrappedSkewBinomialQueue)
	if ok {
		if otherQ.IsEmpty() {
			return true
		} else if b.IsEmpty() {
			return false
		}
		return b.Peek().LessThan(otherQ.Peek())
	}
	return false
}

func skewQToBootstrappedQ(q SkewBinomialQueue) BootstrappedSkewBinomialQueue {
	if q.IsEmpty() {
		return NewEmptyBootstrappedSkewBinomialQueue()
	}
	queuePriority, higherLevelPrimitiveQ := q.Dequeue()
	bootstrappedQ := queuePriority.(BootstrappedSkewBinomialQueue)

	var highestPriorityObject QueuePriority
	highestPriorityObject, bootstrappedQ = bootstrappedQ.Dequeue()
	higherLevelPrimitiveQ = higherLevelPrimitiveQ.Enqueue(bootstrappedQ)

	skewQ := higherLevelPrimitiveQ.(SkewBinomialQueue)
	return BootstrappedSkewBinomialQueue{
		highestPriorityObject: highestPriorityObject,
		priorityQueue:         &skewQ,
		length:                q.Length(),
	}
}
