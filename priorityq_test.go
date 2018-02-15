package priorityq

import (
	"fmt"
	//"math/rand"
	"testing"
	"time"
	"unsafe"
)

type IntegerPriorityScorer struct {
	value int
}

func (i IntegerPriorityScorer) String() string {
	return fmt.Sprintf("<Integer value %d>", i.value)
}

const TEST_TIME = true

func (i IntegerPriorityScorer) Score() int {
	return i.value
}

func TestEnqueueLength(t *testing.T) {
	q := newEmptyBootstrappedSkewBinomial()
	if q.Length() != 0 {
		t.Error("Queue length is not 0")
	}
	q = q.Enqueue(
		IntegerPriorityScorer{0},
	)
	if q.Length() != 1 {
		t.Error("Queue length is not 1")
	}
	_, q = q.Dequeue()
	if q.Length() != 0 {
		t.Error("Queue length is not 0")
	}
}

func TestEnqueueDequeue(t *testing.T) {
	return
	q := newEmptyBootstrappedSkewBinomial()
	values := []int{20, 10, 30, 5, 3, 0, 25}

	for _, value := range values {
		q = q.Enqueue(
			IntegerPriorityScorer{value},
		)
	}
	dequeueValues := []int{}
	var queuePriority PriorityScorer
	for {
		queuePriority, q = q.Dequeue()
		dequeued, ok := queuePriority.(IntegerPriorityScorer)
		if !ok {
			break
		}
		dequeueValues = append(dequeueValues, dequeued.value)
	}
	expectedValues := []int{0, 3, 5, 10, 20, 25, 30}
	for index := range dequeueValues {
		if dequeueValues[index] != expectedValues[index] {
			t.Error("Values not equal")
		}
	}

}

func TestMeld(t *testing.T) {
	return
	q1 := newEmptyBootstrappedSkewBinomial()
	values := []int{1, 2, 3}
	for _, value := range values {
		q1 = q1.Enqueue(
			IntegerPriorityScorer{value},
		)
	}

	q2 := newEmptyBootstrappedSkewBinomial()
	values = []int{4, 5, 6}
	for _, value := range values {
		q2 = q2.Enqueue(
			IntegerPriorityScorer{value},
		)
	}
	q3 := q1.Meld(q2)
	dequeueValues := []int{}
	expectedLength := 6
	if q3.Length() != expectedLength {
		t.Error("Lengths are not equal")
	}
	var queuePriority PriorityScorer
	for {
		queuePriority, q3 = q3.Dequeue()
		dequeued, ok := queuePriority.(IntegerPriorityScorer)
		if !ok {
			break
		}
		dequeueValues = append(dequeueValues, dequeued.value)
	}
	expectedValues := []int{1, 2, 3, 4, 5, 6}
	for index := range dequeueValues {
		if dequeueValues[index] != expectedValues[index] {
			t.Error("Values not equal")
		}
	}
}

func TestIsEmpty(t *testing.T) {
	q := newEmptyBootstrappedSkewBinomial()
	if !q.IsEmpty() {
		t.Error("Queue is not empty")
	}
	q = q.Enqueue(
		IntegerPriorityScorer{0},
	)
	if q.IsEmpty() {
		t.Error("Queue is empty")
	}
}

func int64LessThan(i1, i2 unsafe.Pointer) bool {
	return *(*int64)(i1) < *(*int64)(i2)
}

func TestThreadSafetyListInsert(t *testing.T) {
	return
	list := ThreadSafeList{}
	var randomNumbers []int64
	sampleSize := 1000
	// var seed int64 = 10
	// r1 := rand.New(rand.NewSource(seed))
	for i := 0; i < sampleSize; i++ {
		// randomNumbers = append(randomNumbers, int64(r1.Intn(sampleSize)))

		randomNumbers = append(randomNumbers, int64(i))
	}
	for _, number := range randomNumbers {
		go func(num int64) {
			list.InsertObject(unsafe.Pointer(&num), int64LessThan)
		}(number)
	}

	time.Sleep(1 * time.Second)

	if list.Count() != sampleSize {
		t.Error("Size of list not accurate")
	}
	for i := 0; i < sampleSize; i++ {
		go func(index int) {
			list.PopHead()
		}(i)
	}
	time.Sleep(1 * time.Second)

	if list.Count() != 0 {
		t.Error("Thread safe pop head did not work")
	}
}

func TestListInsertObject(t *testing.T) {
	list := ThreadSafeList{}
	items := []int64{30, 10, 2, 4, 17, 5, 20}
	expectedItems := []int64{2, 4, 5, 10, 17, 20, 30}

	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}
	var sortedItems []int64
	for sortedItem := range list.Iter() {
		sortedItems = append(sortedItems, *(*int64)(sortedItem))
	}
	for index := range sortedItems {
		if sortedItems[index] != expectedItems[index] {
			t.Error("Values not equal")
		}
	}
}

func TestPopFirst(t *testing.T) {

	list := ThreadSafeList{}

	// test empty
	failAddress := unsafe.Pointer(new(int))

	result := list.PopFirst(failAddress)
	if result != failAddress {
		t.Error("Failure for empty case, should have gotten fail address")
	}

	items := []int64{30, 10, 2, 4, 17, 20}

	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}

	result = list.PopFirst(unsafe.Pointer(failAddress))
	poppedValue := *(*int64)(result)

	if poppedValue != 2 {
		t.Error("Unexpected value for popped head", poppedValue)
	}
}

func TestPopNth(t *testing.T) {
	list := ThreadSafeList{}

	items := []int64{30, 10, 2, 4, 17, 20}

	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}

	// test empty
	failAddress := new(int)
	result := list.PopNth(100, unsafe.Pointer(failAddress))

	if (*int)(result) != failAddress {
		t.Error("Fallback value does not work")
	}

	result = list.PopNth(3, unsafe.Pointer(failAddress))
	poppedValue := *(*int64)(result)
	if poppedValue != 10 {
		t.Error("Unexpected popped value", poppedValue)
	}
	result = list.PopNth(3, unsafe.Pointer(failAddress))
	poppedValue = *(*int64)(result)
	if poppedValue != 17 {
		t.Error("Unexpected popped value", poppedValue)
	}

}
func TestListPopHead(t *testing.T) {
	list := ThreadSafeList{}
	items := []int64{30, 10, 2, 4, 17, 5, 20}

	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}
	poppedItem := list.PopHead()
	poppedValue := *(*int64)(poppedItem)
	if poppedValue != 2 {
		t.Error("Pop failed")
	}
	poppedItem = list.PopHead()
	poppedValue = *(*int64)(poppedItem)
	if poppedValue != 4 {
		t.Error("Pop failed")
	}
}

func TestListDeleteObject(t *testing.T) {
	return
	list := ThreadSafeList{}
	items := []int64{30, 10, 2, 4, 17, 5, 20}

	var addrToDelete *int64
	for _, item := range items {
		newlyAllocatedItem := item
		if item == 2 {
			addrToDelete = &newlyAllocatedItem
		}
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}
	list.DeleteObject(unsafe.Pointer(addrToDelete))
	poppedItem := list.PopHead()
	poppedValue := *(*int64)(poppedItem)
	if poppedValue != 4 {
		t.Error("Delete failed")
	}
}

func TestListCounter(t *testing.T) {
	list := ThreadSafeList{}
	items := []int64{30, 10, 2, 4, 17, 5, 20}
	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}
	if len(items) != list.Count() {
		t.Error("Count not equal")
	}
}
func TestListIter(t *testing.T) {
	list := ThreadSafeList{}
	items := []int64{30, 10, 2, 4, 17, 5, 20}
	expectedItems := []int64{2, 4, 5, 10, 17, 20, 30}

	for _, item := range items {
		newlyAllocatedItem := item
		list.InsertObject(unsafe.Pointer(&newlyAllocatedItem), int64LessThan)
	}
	var sortedItems []int64
	for sortedItem := range list.Iter() {
		sortedItems = append(sortedItems, *(*int64)(sortedItem))
	}
	for index := range sortedItems {
		if sortedItems[index] != expectedItems[index] {
			t.Error("Values not equal")
		}
	}
}

func TestEffectiveDequeue(t *testing.T) {
	sampleSize := 100
	var randomNumbers []int
	for i := 0; i < sampleSize; i++ {
		randomNumbers = append(randomNumbers, i)
	}
	q := newEmptySkewBinomial()
	for _, number := range randomNumbers {
		q = q.Enqueue(
			IntegerPriorityScorer{number},
		).(skewBinomial)
	}
	shouldBeSorted := []int{}
	var priority PriorityScorer
	var qP PriorityQ
	for {
		priority, qP = q.Dequeue()
		q = qP.(skewBinomial)
		intPriority, ok := priority.(IntegerPriorityScorer)
		if ok {
			shouldBeSorted = append(shouldBeSorted, intPriority.value)
			validateSortedList(shouldBeSorted, t)
			return
			// successful dequeue
		} else {
			// reached empty queue
			break
		}
	}
}
func TestSpeed(t *testing.T) {
	if !TEST_TIME {
		return
	}

	var randomNumbers []int
	sampleSize := 1000
	//var seed int64 = 10
	//r1 := rand.New(rand.NewSource(seed))
	for i := 0; i < sampleSize; i++ {
		// randomNumbers = append(randomNumbers, r1.Intn(sampleSize))
		randomNumbers = append(randomNumbers, i)
	}

	q := newEmptyBootstrappedSkewBinomial()
	for _, number := range randomNumbers {
		q = q.Enqueue(
			IntegerPriorityScorer{number},
		)
	}

	//shouldBeSorted := []int{}
	var priority PriorityScorer
	for {
		priority, q = q.Dequeue()
		_, ok := priority.(IntegerPriorityScorer)
		if ok {
			//shouldBeSorted = append(shouldBeSorted, intPriority.value)
			//validateSortedList(shouldBeSorted, t)
			// successful dequeue
		} else {
			// reached empty queue
			break
		}
	}
	/*
		if len(shouldBeSorted) != sampleSize {
			t.Error("length of dequeued data is not equal to # items enqueued")
		}
	*/
}

func validateSortedList(shouldBeSorted []int, t *testing.T) {
	for index := 0; index < len(shouldBeSorted)-1; index++ {
		previous := index
		current := index + 1
		if shouldBeSorted[current]-shouldBeSorted[previous] > 1 {
			t.Error(
				"Missing value in list",
				shouldBeSorted[current],
				shouldBeSorted[previous],
			)
			panic("stop")
		}
		if shouldBeSorted[current] == shouldBeSorted[previous] {
			t.Error(
				"duplicate values in list",
				shouldBeSorted[current],
				shouldBeSorted[previous],
			)
			panic("stop")
		}
		if shouldBeSorted[current] < shouldBeSorted[previous] {
			t.Error("data is not sorted")
			panic("stop")
		}
	}
}

func TestSpeedFreeList(t *testing.T) {
	if !TEST_TIME {
		return
	}
	var randomNumbers []int
	sampleSize := 1000000
	//var seed int64 = 10
	//r1 := rand.New(rand.NewSource(seed))
	for i := 0; i < sampleSize; i++ {
		// randomNumbers = append(randomNumbers, r1.Intn(sampleSize))
		randomNumbers = append(randomNumbers, i)
	}

	start := time.Now()
	q := NewMutableParallelQ()
	for index, number := range randomNumbers {
		if index%10000 == 0 {
			percentDone := 100.0 * (float64(index) / float64(sampleSize))
			fmt.Printf("%f complete, added %d items\n", percentDone, index)
		}
		q = q.Enqueue(
			IntegerPriorityScorer{number},
		)
	}
	fmt.Printf("Enqueued %d total items\n", len(randomNumbers))
	fmt.Printf("ALL DONE HERE\n")
	end := time.Now()
	fmt.Printf("Enqueue delta time: %s\n", end.Sub(start))

	// SBL for some reason if there are no pending operations that's when
	// things seem to be in a bad state
	//(q.(ParallelQ)).BlockUntilNoPending()

	start = time.Now()
	dequeueCount := 0
	var priority PriorityScorer
	for {
		priority, q = q.Dequeue()
		_, ok := priority.(IntegerPriorityScorer)
		if ok {
			dequeueCount++
			//fmt.Printf("Value of int is %s\n", intPriority)
		} else {
			break
		}
	}
	end = time.Now()
	fmt.Printf("Dequeued %d total items in time: %s\n", dequeueCount, end.Sub(start))
	/*
		var priority PriorityScorer
		for {
			priority, q = q.Dequeue()
			//fmt.Printf("Dequeued object: %s\n", priority)
			something, ok := priority.(IntegerPriorityScorer)

			_, ahfuck := priority.(bootstrappedSkewBinomial)
			if ahfuck {
				panic("DAMMIT, BOOTSTRAPPED SKEWS ARE BEING INSERTED INTO OTHER BOOTSTRAPPED SKEWS")
			}
			if ok {
				// successful dequeue
				dequeueCount++
			} else {
				fmt.Printf("Value of something was %s\n", something)
				fmt.Printf("Stopping after dequeueing %d items\n", dequeueCount)
				// reached empty queue
				break
			}
		}
		time.Sleep(1 * time.Second)
		fmt.Printf("TRYING TO DEQ again\n")
		priority, q = q.Dequeue()
		_, ok := priority.(IntegerPriorityScorer)
		if ok {
			fmt.Printf("YOU SUCK AT PROGRAMMING\n")
		}
	*/
}
