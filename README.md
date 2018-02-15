# Thread-Safe Lock Free Priority Queue

## Background
This project is based on Dr. Chris Okasaki's paper, [Optimal Purely Functional Priority Queues](http://www.brics.dk/RS/96/37/BRICS-RS-96-37.pdf). You can also read [my own blog](http://scottlobdell.me/2016/09/thread-safe-lock-free-priority-queues-golang/) for background, motivation, and analysis of the repository. The underlying fundamental data structure in this repository is the Bootstrapped Skew Binomial Queue, which is outlined in Dr. Okasaki's paper. At one abstraction level higher, a separate data structure, concisely labeled as a "Parallel Queue", exists that enables thread-safe (and lock free!) concurrent dequeue operations at the cost of sacrificing absolute correctness.

The motivation behind the thread-safe version of the datastructure stemmed from a generally pragmatic use case of priority queues in which:
* thread-safe concurrent access is desirable
* returning the absolute highest priority item in a saturated environment is not necessarily required

It's also worth noting that a skip list would provide a generally comparable implementation of a priority queue, but enqueue() and dequeue() are swapped in terms of run-time complexity between O(1) and O(log N). However, the priority queues implemented here provide the distinct advantage that meld() is supported as a constant time operation, making it ideal for any environment where separate priority queues are routinely merged.

## API
For the TLDR, check out [public.go](https://github.com/slobdell/skew-binomial-queues/blob/master/public.go) to understand the access and instantiation patterns of the data structures provided.

## General Time Complexity
#### Enqueue: O(1)
#### Peek: O(1)
#### Dequeue: O(log N)
#### Length: O(1)
#### Meld: O(1)

## Usage:
```
type ArbitraryDataType struct {
    ArbitraryData string
    score         int
}

func (a ArbitraryDataType) Score() int {
  return a.score
}

arbitraryScore := 100
q1 := priorityQ.NewImmutableSynchronousQ()

/**********  ENQUEUE  **********/
q1 = q1.Enqueue(
    ArbitraryDataType{
        "some_data",
        arbitraryScore,
    }
)


/**********  PEEK  **********/
highestPriorityData := q1.Peek()


/**********  DEQUEUE  **********/
highestPriorityData, q1 = q1.Dequeue()
casted, ok := highestPriorityData.(ArbitraryDataType)


/**********  MELD  **********/
anotherQ := NewEmptyBootstrappedSkewBinomialQueue()
melded := q1.Meld(anotherQ)
```

### All Cases
* Constant time enqueue

### Use Cases for the Mutable Parallel Priority Queue
* Thread safe priority queue
* Constant time dequeue on the hot path
* A relatively high priority element, but not necessarily the highest priority element, is an acceptable value during concurrent dequeue operations
* Best case constant time meld

### Use Cases for the Immutable Synchronous Priority Queue
* Not inherently thread safe
* Constant time meld
* Complete immutability is desirable (Any reference to a particular version of the data structure can be maintained indefinitely)
* Logarithmic time dequeues
