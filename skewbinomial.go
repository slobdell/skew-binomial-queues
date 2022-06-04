package priorityq

import (
	"fmt"
	"math"
	"sort"
)

//5,180,022
var skewHeapCount *int32 = new(int32)

// 1,4595,392
var skewQueueCount *int32 = new(int32)

// 6,526,797
var bootstrappedQueueCount *int32 = new(int32)

func min(x, y PriorityScorer) PriorityScorer {
	if x.LessThan(y) {
		return x
	}
	return y
}

type node interface {
	Rank() int
	Length() int
	Children() []node
	Peek() PriorityScorer
	Link(heaps ...node) node
	IsEmpty() bool
	UnorderedRangeScan(Visitor)
}

var NULL_OBJECT = nullObject{}

type nullObject struct {
	head     *nullObject
	children []node
}

func (n nullObject) Link(heaps ...node) node {
	return n
}

func (n nullObject) Peek() PriorityScorer {
	return nil
}

func (n nullObject) Children() []node {
	return n.children
}

func (n nullObject) Rank() int {
	return -1
}

func (n nullObject) IsEmpty() bool {
	return true
}

func (n nullObject) Length() int {
	return 0
}

func (n nullObject) UnorderedRangeScan(Visitor) {
}

type skewBinomialHeap struct {
	rank     int
	priority PriorityScorer
	children []node // this is possibly problematic but I'm not sure..pointer vs non pointer
	length   int
}

type skewByRank []skewBinomial

func (a skewByRank) Len() int           { return len(a) }
func (a skewByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a skewByRank) Less(i, j int) bool { return a[i].rank() > a[j].rank() }

type byRank []node

func (a byRank) Len() int           { return len(a) }
func (a byRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRank) Less(i, j int) bool { return a[i].Rank() < a[j].Rank() }

type byPriority []node

func (a byPriority) Len() int           { return len(a) }
func (a byPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPriority) Less(i, j int) bool { return a[i].Peek().LessThan(a[j].Peek()) }

func newSkewBinomialHeap(priority PriorityScorer) node {
	return skewBinomialHeap{
		priority: priority,
		length:   1,
	}
}

func highestPriorityHeap(heaps ...node) node {
	sort.Sort(
		byPriority(
			heaps,
		),
	)
	return heaps[0]
}

func lowerRankHeaps(heaps ...node) []node {
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

func newNode(priority PriorityScorer, rank int, children []node) node {
	var existingLength int
	for _, node := range children {
		// TODO see if I can do this more elegantly?
		existingLength += node.Length()
	}
	return skewBinomialHeap{
		rank:     rank,
		priority: priority,
		children: children,
		length:   1 + existingLength,
	}
}

func (s skewBinomialHeap) IsEmpty() bool {
	return false
}

func (s skewBinomialHeap) Children() []node {
	return s.children
}

func (s skewBinomialHeap) Link(heaps ...node) node {
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

func (s skewBinomialHeap) Length() int {
	return s.length
}

func (s skewBinomialHeap) Rank() int {
	return s.rank
}

func (s skewBinomialHeap) Peek() PriorityScorer {
	return s.priority
}

func (s skewBinomialHeap) UnorderedRangeScan(v Visitor) {
	v(s.priority)
	for _, c := range s.children {
		c.UnorderedRangeScan(v)
	}
}

type skewBinomial struct {
	heapHead     node
	rightSibling *skewBinomial
}

func newEmptySkewBinomial() skewBinomial {
	return skewBinomial{
		heapHead:     NULL_OBJECT,
		rightSibling: nil,
	}
}

func newSkewBinomial(heapHead node, rightSibling *skewBinomial) skewBinomial {
	return skewBinomial{
		heapHead:     heapHead,
		rightSibling: rightSibling,
	}
}

func (q skewBinomial) String() string {
	return fmt.Sprintf("<Skew Binomial Queue, peek value: %s, Length: %d>", q.Peek(), q.Length())
}

func (q skewBinomial) firstTwoTreesEqualRank() bool {
	if q.heapHead.IsEmpty() {
		return false
	}
	if q.rightSibling == nil {
		return false
	}
	return q.rank() == q.rightSibling.rank()
}

func (q skewBinomial) Enqueue(priority PriorityScorer) PriorityQ {
	// SBL ADD A FUNCTION TO WALK WHAT THE PRIORITIES ARE...
	if q.firstTwoTreesEqualRank() {
		// null checks should not be necessary here since first two trees are equal
		return newSkewBinomial(
			newSkewBinomialHeap(priority).Link(
				q.heapHead,
				q.rightSibling.heapHead,
			),
			q.rightSibling.rightSibling,
		)
	}
	return newSkewBinomial(
		newSkewBinomialHeap(priority),
		&q,
	)
}

func (q skewBinomial) UnorderedRangeScan(v Visitor) {
	if q.IsEmpty() {
		return
	}
	q.heapHead.UnorderedRangeScan(v)
	// SBL need to check if rightSibling gets visited twice....
	if q.rightSibling != nil {
		q.rightSibling.UnorderedRangeScan(v)
	}
}

func (q skewBinomial) Peek() PriorityScorer {
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

func (q skewBinomial) Score() int64 {
	return q.Peek().Score()
}

func (q skewBinomial) Meld(otherQ PriorityQ) PriorityQ {
	if q.IsEmpty() {
		return otherQ
	} else if otherQ.IsEmpty() {
		return q
	}

	otherSkewQ, ok := otherQ.(skewBinomial)
	if !ok {
		panic("Meld unequal priority queue types")
	}

	allForests := append(q.asIndividualForests(), otherSkewQ.asIndividualForests()...)
	sort.Sort(
		skewByRank(
			allForests,
		),
	)
	finalQ := new(skewBinomial)

	*finalQ = newEmptySkewBinomial()

	for _, qOneTree := range allForests {
		if finalQ.heapHead.IsEmpty() {
			// TODO avoid special casing this if I can
			*finalQ = newSkewBinomial(
				qOneTree.heapHead,
				nil,
			)
		} else if finalQ.rank() == qOneTree.rank() {
			if finalQ.rightSibling == nil {
				*finalQ = newSkewBinomial(
					finalQ.heapHead.Link(qOneTree.heapHead),
					nil,
				)
			} else {
				priorityQ := finalQ.rightSibling.Meld(
					newSkewBinomial(
						finalQ.heapHead.Link(qOneTree.heapHead),
						nil,
					),
				)
				*finalQ, ok = priorityQ.(skewBinomial)
			}
		} else if qOneTree.rank() < finalQ.rank() {
			newQ := newSkewBinomial(
				qOneTree.heapHead,
				finalQ,
			)
			finalQ = new(skewBinomial)
			*finalQ = newQ
		} else {
			*finalQ = newSkewBinomial(
				finalQ.heapHead,
				&qOneTree,
			)
		}
	}
	return *finalQ
}

func (q skewBinomial) rank() int {
	// TODO unsure if this method makes this unclear, but most use cases for checking
	// rank involve cheaking the heap head
	return q.heapHead.Rank()
}

func (q skewBinomial) asIndividualForests() []skewBinomial {
	if q.IsEmpty() {
		return nil
	} else if q.rightSibling == nil {
		return []skewBinomial{q}
	}
	return append(
		[]skewBinomial{newSkewBinomial(q.heapHead, nil)},
		q.rightSibling.asIndividualForests()...,
	)
}

func (q skewBinomial) Dequeue() (PriorityScorer, PriorityQ) {
	return q.DequeueWithMergeCallback(
		q.dequeueMergeCallback,
	)
}

func (q skewBinomial) DequeueWithMergeCallback(callback func([]node, ...*skewBinomial) skewBinomial) (PriorityScorer, PriorityQ) {
	if q.IsEmpty() {
		return nil, q
	}
	poppedQueue, remainingQueue := q.popHighestPriorityQ()
	return poppedQueue.Peek(), callback(poppedQueue.heapHead.Children(), &remainingQueue)
}

func (q skewBinomial) dequeueMergeCallback(childNodes []node, remainingQueues ...*skewBinomial) skewBinomial {
	var childrenRankGreaterThanZero []skewBinomial
	var prioritiesRankZero []PriorityScorer
	remainingQPtr := remainingQueues[0]
	remainingQueue := *remainingQPtr
	for _, skewQ := range remainingQueues[1:] {
		newlyAllocatedItem := skewQ
		remainingQueue = remainingQueue.Meld(*newlyAllocatedItem).(skewBinomial)
	}
	for _, child := range childNodes {
		if child.Rank() > 0 {
			childrenRankGreaterThanZero = append(
				childrenRankGreaterThanZero,
				newSkewBinomial(
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
		finalQueue, _ = finalQueue.Meld(queue).(skewBinomial)
	}
	return finalQueue.bulkInsert(prioritiesRankZero...)
}

func (q skewBinomial) Length() int {
	if q.rightSibling == nil || q.rightSibling.heapHead.IsEmpty() {
		return q.heapHead.Length()
	}
	return q.heapHead.Length() + q.rightSibling.Length()
}

func (q skewBinomial) bulkInsert(priorities ...PriorityScorer) skewBinomial {
	if len(priorities) == 0 {
		return q
	}
	return q.Enqueue(
		priorities[0],
	).(skewBinomial).bulkInsert(priorities[1:]...)
}

func (q skewBinomial) IsEmpty() bool {
	return q.heapHead.IsEmpty()
}

func (q skewBinomial) popHighestPriorityQ() (skewBinomial, skewBinomial) {
	if q.rightSibling == nil {
		return q, newEmptySkewBinomial()
	}
	if q.rightSibling.heapHead.IsEmpty() {
		return newSkewBinomial(
				q.heapHead,
				nil,
			),
			*q.rightSibling
	}
	if q.heapHead.Peek().LessThan(q.rightSibling.Peek()) {
		return newSkewBinomial(
				q.heapHead,
				nil,
			),
			*q.rightSibling
	}
	poppedQueue, remainingQueue := q.rightSibling.popHighestPriorityQ()
	return poppedQueue, newSkewBinomial(q.heapHead, &remainingQueue)
}

type rootedSkewBinomial struct {
	highestPriorityObject PriorityScorer
	priorityQueue         skewBinomial
}

func newRootedSkewBinomial() rootedSkewBinomial {
	return rootedSkewBinomial{
		highestPriorityObject: nil,
		priorityQueue:         newEmptySkewBinomial(),
	}
}

func (r rootedSkewBinomial) IsEmpty() bool {
	return r.highestPriorityObject == nil
}

func (r rootedSkewBinomial) Enqueue(priority PriorityScorer) rootedSkewBinomial {
	if r.IsEmpty() {
		return rootedSkewBinomial{
			highestPriorityObject: priority,
			priorityQueue:         newEmptySkewBinomial(),
		}
	}
	if r.highestPriorityObject.LessThan(priority) {
		return rootedSkewBinomial{
			highestPriorityObject: r.highestPriorityObject,
			priorityQueue:         r.priorityQueue.Enqueue(priority).(skewBinomial),
		}
	}
	return rootedSkewBinomial{
		highestPriorityObject: priority,
		priorityQueue: r.priorityQueue.Enqueue(
			r.highestPriorityObject,
		).(skewBinomial),
	}
}

func (r rootedSkewBinomial) Dequeue() (PriorityScorer, rootedSkewBinomial) {
	if r.priorityQueue.IsEmpty() {
		return r.highestPriorityObject, newRootedSkewBinomial()
	}
	highestPriorityObject, priorityQueue := r.priorityQueue.Dequeue()
	return r.highestPriorityObject, rootedSkewBinomial{
		highestPriorityObject: highestPriorityObject,
		priorityQueue:         priorityQueue.(skewBinomial),
	}
}

func (r rootedSkewBinomial) Meld(otherQ rootedSkewBinomial) rootedSkewBinomial {
	mergedPrimitiveQueue := r.priorityQueue.Meld(otherQ.priorityQueue)
	if r.highestPriorityObject.LessThan(otherQ.highestPriorityObject) {
		return rootedSkewBinomial{
			highestPriorityObject: r.highestPriorityObject,
			priorityQueue: mergedPrimitiveQueue.Enqueue(
				otherQ.highestPriorityObject,
			).(skewBinomial),
		}
	}
	return rootedSkewBinomial{
		highestPriorityObject: otherQ.highestPriorityObject,
		priorityQueue: mergedPrimitiveQueue.Enqueue(
			r.highestPriorityObject,
		).(skewBinomial),
	}
}

func (r rootedSkewBinomial) Peek() PriorityScorer {
	return r.highestPriorityObject
}

func (r rootedSkewBinomial) Length() int {
	if r.IsEmpty() {
		return 0
	}
	return 1 + r.priorityQueue.Length()
}

type bootstrappedSkewBinomial struct {
	highestPriorityObject PriorityScorer
	priorityQueue         *skewBinomial
	length                int
}

func (b bootstrappedSkewBinomial) String() string {
	return fmt.Sprintf("<bootstrappedSkewBinomial, peek value: %d, length: %d>", b.Peek(), b.Length())
}

func (b bootstrappedSkewBinomial) IsEmpty() bool {
	return b.highestPriorityObject == nil
}

func (b bootstrappedSkewBinomial) UnorderedRangeScan(v Visitor) {
	if b.IsEmpty() {
		return
	}
	v(b.highestPriorityObject)
	if b.priorityQueue != nil {
		b.priorityQueue.UnorderedRangeScan(func(otherQ PriorityScorer) {
			casted := otherQ.(bootstrappedSkewBinomial)
			casted.UnorderedRangeScan(v)
		})
	}
}

func (b bootstrappedSkewBinomial) Enqueue(priority PriorityScorer) bootstrappedSkewBinomial {
	newEmptySkewQ := newEmptySkewBinomial()
	return b.Meld(
		bootstrappedSkewBinomial{
			highestPriorityObject: priority,
			priorityQueue:         &newEmptySkewQ,
			length:                1,
		},
	)
}

func (b bootstrappedSkewBinomial) Peek() PriorityScorer {
	return b.highestPriorityObject
}

func (b bootstrappedSkewBinomial) Dequeue() (PriorityScorer, bootstrappedSkewBinomial) {
	return b.DequeueWithMergeCallback(
		b.priorityQueue.dequeueMergeCallback,
	)
}

func (b bootstrappedSkewBinomial) DequeueWithMergeCallback(mergeCallback func([]node, ...*skewBinomial) skewBinomial) (PriorityScorer, bootstrappedSkewBinomial) {
	if b.priorityQueue.IsEmpty() || b.IsEmpty() {
		return b.highestPriorityObject, newEmptyBootstrappedSkewBinomial()
	}
	queuePriority, updatedPrimitiveQueue := b.priorityQueue.DequeueWithMergeCallback(mergeCallback)
	highestPriorityBootstrappedQueue, ok := queuePriority.(bootstrappedSkewBinomial)

	if !ok {
		panic("Cannot cast to a Bootstrapped Queue. This case should not have been reached")
	}
	updatedSkewQ := updatedPrimitiveQueue.(skewBinomial)
	mergedPrimitiveQ := mergeCallback(
		[]node{},
		&updatedSkewQ,
		highestPriorityBootstrappedQueue.priorityQueue,
	)

	updatedBootstrappedQueue := bootstrappedSkewBinomial{
		highestPriorityObject: highestPriorityBootstrappedQueue.highestPriorityObject,
		priorityQueue:         &mergedPrimitiveQ,
		length:                b.Length() - 1, // FIXME this no longer works w callback, value is wrong
	}
	return b.highestPriorityObject, updatedBootstrappedQueue
}

func (b bootstrappedSkewBinomial) Length() int {
	return b.length
}

func (b bootstrappedSkewBinomial) Meld(otherQ bootstrappedSkewBinomial) bootstrappedSkewBinomial {
	if b.IsEmpty() {
		return otherQ
	} else if otherQ.IsEmpty() {
		return b
	} else if b.Peek().LessThan(otherQ.Peek()) {
		priorityQueue := b.priorityQueue.Enqueue(
			otherQ,
		).(skewBinomial)
		return bootstrappedSkewBinomial{
			highestPriorityObject: b.Peek(),
			priorityQueue:         &priorityQueue,
			length:                b.Length() + otherQ.Length(),
		}
	}
	primitiveQueue := otherQ.priorityQueue.Enqueue(
		b,
	).(skewBinomial)

	return bootstrappedSkewBinomial{
		highestPriorityObject: otherQ.Peek(),
		priorityQueue:         &primitiveQueue,
		length:                b.Length() + otherQ.Length(),
	}
}

func (b bootstrappedSkewBinomial) Score() int64 {
	if b.IsEmpty() {
		return math.MaxInt64
	}
	return b.Peek().Score()
}

func (b bootstrappedSkewBinomial) LessThan(other PriorityScorer) bool {
	if b.IsEmpty() {
		return false
	}
	casted := other.(bootstrappedSkewBinomial)
	if casted.IsEmpty() {
		return true
	}
	return b.Peek().LessThan(casted.Peek())
}

func newEmptyBootstrappedSkewBinomial() bootstrappedSkewBinomial {
	newEmptySkewQ := newEmptySkewBinomial()
	return bootstrappedSkewBinomial{
		highestPriorityObject: nil,
		priorityQueue:         &newEmptySkewQ,
		length:                0,
	}
}

func skewQToBootstrappedQ(q skewBinomial) bootstrappedSkewBinomial {
	if q.IsEmpty() {
		return newEmptyBootstrappedSkewBinomial()
	}
	queuePriority, higherLevelPrimitiveQ := q.Dequeue()
	bootstrappedQ := queuePriority.(bootstrappedSkewBinomial)

	var highestPriorityObject PriorityScorer
	highestPriorityObject, bootstrappedQ = bootstrappedQ.Dequeue()
	higherLevelPrimitiveQ = higherLevelPrimitiveQ.Enqueue(bootstrappedQ)

	skewQ := higherLevelPrimitiveQ.(skewBinomial)
	return bootstrappedSkewBinomial{
		highestPriorityObject: highestPriorityObject,
		priorityQueue:         &skewQ,
		length:                q.Length(),
	}
}
