package jiffy

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

const (
	bufferSize = 1620

	setStateEmpty    = 0
	setStateFull     = 1
	setStateConsumed = 2
)

type node struct {
	data interface{}
	// isSet is an int32 because we need an atomic type to represent a bool, and
	// int32 is the smallest atomic type
	//
	// 0: empty: data is not done being added
	// 1: set: data is done being added, ready to be consumed
	// 2: handled: data has been consumed
	isSet int32
}

type buffer struct {
	curr     []*node
	head     uint64
	position uint64

	// next is an atomic pointer for the next buffer
	next     unsafe.Pointer
	previous *buffer
}

type Queue struct {
	head *buffer
	// tailBuffer is an atomic pointer for the tail of the queue
	tailBuffer unsafe.Pointer
	// tail is an atomic uint64 for the tail index
	tail uint64
}

func New() *Queue {
	q := &Queue{
		head: &buffer{
			curr:     make([]*node, bufferSize),
			position: 1,
		},
	}
	q.tailBuffer = unsafe.Pointer(&q.head)
	return q
}

func (q *Queue) Add(v interface{}) {
	location := atomic.AddUint64(&q.tail, 1)
	isLastBuffer := true
	tempTail := (*buffer)(atomic.LoadPointer(&q.tailBuffer))
	numElements := bufferSize * tempTail.position

	fmt.Println(tempTail)
	fmt.Println(location)
	fmt.Println(numElements)

	for location >= numElements {
		if atomic.LoadPointer(&tempTail.next) == nil {
			newArr := unsafe.Pointer(&buffer{
				curr:     make([]*node, bufferSize),
				position: tempTail.position + 1,
				previous: tempTail,
			})
			if atomic.CompareAndSwapPointer(&tempTail.next, nil, newArr) {
				atomic.CompareAndSwapPointer(&q.tailBuffer, unsafe.Pointer(tempTail), newArr)
			}
		}
		tempTail = (*buffer)(atomic.LoadPointer(&q.tailBuffer))
		numElements = bufferSize * tempTail.position
	}

	// calculating the amount of item in the queue - the current buffer
	prevSize := bufferSize * (tempTail.position - 1)

	for location != prevSize {
		// location is in a previous buffer from the buffer pointed by tail
		tempTail = tempTail.previous
		prevSize = bufferSize * (tempTail.position - 1)
		isLastBuffer = false
	}

	// location is in this buffer
	n := tempTail.curr[location-prevSize]

	if atomic.LoadInt32(&n.isSet) == setStateEmpty {
		n.data = v
		atomic.StoreInt32(&n.isSet, setStateFull)
		if location-prevSize == 1 && isLastBuffer {
			// allocating a new buffer and adding it to the queue
			newArr := unsafe.Pointer(&buffer{
				curr:     make([]*node, bufferSize),
				position: tempTail.position + 1,
				previous: tempTail,
			})
			atomic.CompareAndSwapPointer(&tempTail.next, nil, newArr)
		}
	}
}

func (q *Queue) Get() interface{} {
	return nil
}
