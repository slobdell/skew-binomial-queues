# Thread-Safe Lock Free Priority Queue

The `BootstrappedSkewBinomialQueue` provides the fundamental data structure specific to this repository. It is based on Dr. Chris Okasaki's paper, [Optimal Purely Functional Priority Queues](http://www.brics.dk/RS/96/37/BRICS-RS-96-37.pdf). You can also read [my own blog](http://scottlobdell.me/2016/09/thread-safe-lock-free-priority-queues-golang/) for background, motivation, and analysis of the repository.

The supported operations and their associated run time complexities are as follows:

#### Enqueue: O(1)
#### Peek: O(1)
#### Dequeue: O(log N)
#### Length: O(1)
#### Meld: O(1)

The fundamental advantage of a comparable priority queue datastructure is the ability to meld queues.


## Usage:
```
type ArbitraryDataType struct {
    ArbitraryData string
    Score         int
}

func (a ArbitraryDataType) LessThan(otherPriority skewBinomialQ.QueuePriority) bool {
	casted, ok := otherPriority.(ArbitraryDataType)
	if ok {
		return a.Score < casted.Score
	}
	return false
}

arbitraryScore := 100
q1 := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()

/**********  ENQUEUE  **********/
q1 = q.Enqueue(
    ArbitraryDataType{
        "some_data",
        arbitraryScore,
    }
)


/**********  PEEK  **********/
highestPriorityData := q1.Peek()


/**********  DEQUEUE  **********/
highestPriorityData, ok = q1.Dequeue()
casted, ok := highestPriorityData.(ArbitraryDataType)


/**********  MELD  **********/
anotherQ := NewEmptyBootstrappedSkewBinomialQueue()
melded := q1.Meld(anotherQ)
```

The library was also an experiment with lock-free datastructures. You can also initialize a `LazyMergeSkewBinomialQueue` in place of a `BootstrappedSkewBinomialQueue`. This datastructure will allow for thread-safe operations, but it will NOT guarantee that the highest priority element is popped in the case that the queue is saturated with dequeues from multiple threads.

As the code currently stands, there are some less than optimal uses of goroutines that need to be cleaned up, so it is unlikely that the concurrent implementation offers significant benefit to any consumers.
