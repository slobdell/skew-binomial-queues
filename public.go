package priorityq

type PriorityQ interface {
	Enqueue(priority PriorityScorer) PriorityQ
	Peek() PriorityScorer
	Meld(otherQ PriorityQ) PriorityQ
	Dequeue() (PriorityScorer, PriorityQ)
	Length() int
	IsEmpty() bool
}

type PriorityScorer interface {
	Score() int64
}

func NewMutableParallelQ() PriorityQ {
	return newParallelQ()
}

func NewImmutableSynchronousQ() PriorityQ {
	return newSynchronousQ()
}
