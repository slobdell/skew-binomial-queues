package priorityq

type SynchronousQ struct {
	bootstrappedSkewBinomial
}

func (s SynchronousQ) Dequeue() (PriorityScorer, PriorityQ) {
	priority, bootstrappedQ := s.bootstrappedSkewBinomial.Dequeue()
	return priority, SynchronousQ{
		bootstrappedSkewBinomial: bootstrappedQ,
	}
}

func (s SynchronousQ) Enqueue(priority PriorityScorer) PriorityQ {
	bootstrappedQ := s.bootstrappedSkewBinomial.Enqueue(priority)
	return SynchronousQ{
		bootstrappedSkewBinomial: bootstrappedQ,
	}
}

func (s SynchronousQ) Meld(otherQ PriorityQ) PriorityQ {
	casted := otherQ.(SynchronousQ)
	melded := s.bootstrappedSkewBinomial.Meld(casted.bootstrappedSkewBinomial)
	return SynchronousQ{
		bootstrappedSkewBinomial: melded,
	}
}

func newSynchronousQ() SynchronousQ {
	return SynchronousQ{}
}
