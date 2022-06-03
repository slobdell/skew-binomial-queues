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
	// Score provides a default mechanism to prioritize items;
	// LessThan can be implemented with a simple Score() < other.Score()
	Score() int64

	// LessThan can generally be implemented as Score() < other.Score()
	// However, this does not work for comparing items such as strings,
	// and therefore this method was introduced
	LessThan(other PriorityScorer) bool
}

func NewMutableParallelQ() PriorityQ {
	return newParallelQ()
}

func NewImmutableSynchronousQ() PriorityQ {
	return newSynchronousQ()
}
