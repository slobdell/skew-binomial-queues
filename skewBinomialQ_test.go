package skewBinomialQ_test

import (
	"math/rand"
	"skewBinomialQ"
	"testing"
	"unsafe"
)

type IntegerQueuePriority struct {
	value int
}

func (i IntegerQueuePriority) LessThan(otherPriority skewBinomialQ.QueuePriority) bool {
	integerQueuePriority, ok := otherPriority.(IntegerQueuePriority)
	if ok {
		return i.value < integerQueuePriority.value
	}
	return false
}

func TestEnqueueLength(t *testing.T) {
	q := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	if q.Length() != 0 {
		t.Error("Queue length is not 0")
	}
	q = q.Enqueue(
		IntegerQueuePriority{0},
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
	q := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	values := []int{20, 10, 30, 5, 3, 0, 25}

	for _, value := range values {
		q = q.Enqueue(
			IntegerQueuePriority{value},
		)
	}
	dequeueValues := []int{}
	var queuePriority skewBinomialQ.QueuePriority
	for {
		queuePriority, q = q.Dequeue()
		dequeued, ok := queuePriority.(IntegerQueuePriority)
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
	q1 := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	values := []int{1, 2, 3}
	for _, value := range values {
		q1 = q1.Enqueue(
			IntegerQueuePriority{value},
		)
	}

	q2 := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	values = []int{4, 5, 6}
	for _, value := range values {
		q2 = q2.Enqueue(
			IntegerQueuePriority{value},
		)
	}
	q3 := q1.Meld(q2)
	dequeueValues := []int{}
	expectedLength := 6
	if q3.Length() != expectedLength {
		t.Error("Lengths are not equal")
	}
	var queuePriority skewBinomialQ.QueuePriority
	for {
		queuePriority, q3 = q3.Dequeue()
		dequeued, ok := queuePriority.(IntegerQueuePriority)
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
	q := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	if !q.IsEmpty() {
		t.Error("Queue is not empty")
	}
	q = q.Enqueue(
		IntegerQueuePriority{0},
	)
	if q.IsEmpty() {
		t.Error("Queue is empty")
	}
}

func int64LessThan(i1, i2 unsafe.Pointer) bool {
	return *(*int64)(i1) < *(*int64)(i2)
}

func TestListInsertObject(t *testing.T) {
	list := skewBinomialQ.ThreadSafeList{}
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

func TestListPopHead(t *testing.T) {
	list := skewBinomialQ.ThreadSafeList{}
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
	list := skewBinomialQ.ThreadSafeList{}
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

func TestListIter(t *testing.T) {
	list := skewBinomialQ.ThreadSafeList{}
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

func TestSpeed(t *testing.T) {
	/* perhaps disable in some cases...test speed for various implementations */

	var randomNumbers []int
	sampleSize := 100
	var seed int64 = 10
	r1 := rand.New(rand.NewSource(seed))
	for i := 0; i < sampleSize; i++ {
		randomNumbers = append(randomNumbers, r1.Intn(sampleSize))
	}

	q := skewBinomialQ.NewEmptyBootstrappedSkewBinomialQueue()
	for _, number := range randomNumbers {
		q = q.Enqueue(
			IntegerQueuePriority{number},
		)
	}

	var priority skewBinomialQ.QueuePriority
	for {
		priority, q = q.Dequeue()
		_, ok := priority.(IntegerQueuePriority)
		if ok {
			// successful dequeue
		} else {
			// reached empty queue
			break
		}
	}
}

func TestFreeListQueue(t *testing.T) {
	skewBinomialQ.NewFreeListQueue()
}
