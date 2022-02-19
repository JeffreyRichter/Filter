package collections

// Queue is a FIFO queue
type Queue[T any] []T

// Enqueue adds a T to the end of the queue
func (q *Queue[T]) Enqueue(v T) {
	*q = append(*q, v) // Simply append to enqueue
}

// Dequeue removes the T at the front of the queue
func (q *Queue[T]) Dequeue() T {
	v := (*q)[0] // The first element is the one to be dequeued.
	*q = (*q)[1:]
	return v
}

// Stack is a LIFO data structure
type Stack[T any] []T

// Pop removes the T at the top of the stack
func (s *Stack[T]) Pop() T {
	if len(*s) == 0 {
		panic("Attempt to pop from empty stack")
	}
	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return v
}

// Push adds A T to the top of the stack
func (s *Stack[T]) Push(v T) {
	*s = append(*s, v)
}

// Peek returns the T at the top of the stack
func (s *Stack[T]) Peek() (t T, ok bool) {
	if len(*s) == 0 {
		return t, false // Attempt to peek from empty stack
	}
	return (*s)[0], true
}

// Length returns the amount of tokens in the stack
func (s *Stack[T]) Empty() bool {
	return len(*s) == 0
}
