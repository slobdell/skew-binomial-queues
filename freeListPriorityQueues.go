package skewBinomialQ

type FreeListQueue struct {
	SkewBinomialQueue
}

func NewFreeListQueue() FreeListQueue {
	return FreeListQueue{
		heapHead:     NullObject{},
		rightSibling: nil,
	}
}
