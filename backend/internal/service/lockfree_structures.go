package service

import (
	"sync/atomic"
	"unsafe"
)

type LockFreeQueue[T any] struct {
	head atomic.Pointer[node[T]]
	tail atomic.Pointer[node[T]]
}

type node[T any] struct {
	value T
	next  atomic.Pointer[node[T]]
}

func NewLockFreeQueue[T any]() *LockFreeQueue[T] {
	dummy := &node[T]{}
	return &LockFreeQueue[T]{
		head: atomic.Pointer[node[T]]{},
		tail: atomic.Pointer[node[T]]{},
	}
}

func (q *LockFreeQueue[T]) Init() {
	dummy := &node[T]{}
	q.head.Store(dummy)
	q.tail.Store(dummy)
}

func (q *LockFreeQueue[T]) Enqueue(value T) bool {
	newNode := &node[T]{value: value}

	for {
		tail := q.tail.Load()
		next := tail.next.Load()

		if tail == q.tail.Load() {
			if next == nil {
				if tail.next.CompareAndSwap(next, newNode) {
					q.tail.CompareAndSwap(tail, newNode)
					return true
				}
			} else {
				q.tail.CompareAndSwap(tail, next)
			}
		}
	}
}

func (q *LockFreeQueue[T]) Dequeue() (T, bool) {
	for {
		head := q.head.Load()
		tail := q.tail.Load()
		next := head.next.Load()

		if head == q.head.Load() {
			if head == tail {
				if next == nil {
					var zero T
					return zero, false
				}
				q.tail.CompareAndSwap(tail, next)
			} else {
				if next != nil {
					value := next.value
					if q.head.CompareAndSwap(head, next) {
						return value, true
					}
				}
			}
		}
	}
}

func (q *LockFreeQueue[T]) IsEmpty() bool {
	head := q.head.Load()
	tail := q.tail.Load()
	next := head.next.Load()
	return head == tail && next == nil
}

type LockFreeStack[T any] struct {
	top atomic.Pointer[node[T]]
}

func NewLockFreeStack[T any]() *LockFreeStack[T] {
	return &LockFreeStack[T]{}
}

func (s *LockFreeStack[T]) Push(value T) {
	newNode := &node[T]{value: value}
	for {
		newNode.next.Store(s.top.Load())
		if s.top.CompareAndSwap(newNode.next.Load(), newNode) {
			return
		}
	}
}

func (s *LockFreeStack[T]) Pop() (T, bool) {
	for {
		top := s.top.Load()
		if top == nil {
			var zero T
			return zero, false
		}

		next := top.next.Load()
		if s.top.CompareAndSwap(top, next) {
			return top.value, true
		}
	}
}

func (s *LockFreeStack[T]) IsEmpty() bool {
	return s.top.Load() == nil
}

type LockFreeMap[K comparable, V any] struct {
	buckets atomic.Value
	numBuckets int
}

type bucket[K comparable, V any] struct {
	items map[K]V
	mu    unsafe.Pointer[bucketMutex]
}

type bucketMutex struct {
	_ noCopy
	locked bool
}

type noCopy struct{}

func NewLockFreeMap[K comparable, V any](numBuckets int) *LockFreeMap[K, V] {
	if numBuckets <= 0 {
		numBuckets = 256
	}

	buckets := make([]*bucket[K, V], numBuckets)
	for i := 0; i < numBuckets; i++ {
		buckets[i] = &bucket[K, V]{
			items: make(map[K]V),
			mu:    new(unsafe.Pointer[bucketMutex]),
		}
	}

	m := &LockFreeMap[K, V]{
		numBuckets: numBuckets,
	}
	m.buckets.Store(buckets)
	return m
}

func (m *LockFreeMap[K, V]) getBucket(key K) *bucket[K, V] {
	buckets := m.buckets.Load().([]*bucket[K, V])
	hash := hashKey(key)
	return buckets[hash%len(buckets)]
}

func (m *LockFreeMap[K, V]) Set(key K, value V) {
	b := m.getBucket(key)
	for {
		mu := (*bucketMutex)(b.mu.Load())
		if mu == nil || !mu.locked {
			b.items[key] = value
			return
		}
	}
}

func (m *LockFreeMap[K, V]) Get(key K) (V, bool) {
	b := m.getBucket(key)
	for {
		mu := (*bucketMutex)(b.mu.Load())
		if mu == nil || !mu.locked {
			value, ok := b.items[key]
			return value, ok
		}
	}
}

func (m *LockFreeMap[K, V]) Delete(key K) bool {
	b := m.getBucket(key)
	for {
		mu := (*bucketMutex)(b.mu.Load())
		if mu == nil || !mu.locked {
			if _, ok := b.items[key]; ok {
				delete(b.items, key)
				return true
			}
			return false
		}
	}
}

func hashKey[K comparable](key K) int {
	switch v := any(key).(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case string:
		return hashString(v)
	default:
		return 0
	}
}

func hashString(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int(s[i])
	}
	return h
}

type LockFreeList[T any] struct {
	head    atomic.Pointer[node[T]]
	length  atomic.Int64
}

func NewLockFreeList[T any]() *LockFreeList[T] {
	dummy := &node[T]{}
	return &LockFreeList[T]{
		head: atomic.Pointer[node[T]]{},
	}
}

func (l *LockFreeList[T]) Init() {
	dummy := &node[T]{}
	l.head.Store(dummy)
	l.length.Store(0)
}

func (l *LockFreeList[T]) Insert(value T) bool {
	newNode := &node[T]{value: value}

	for {
		head := l.head.Load()
		newNode.next.Store(head)
		if l.head.CompareAndSwap(head, newNode) {
			l.length.Add(1)
			return true
		}
	}
}

func (l *LockFreeList[T]) Remove() (T, bool) {
	for {
		head := l.head.Load()
		next := head.next.Load()

		if next == nil {
			var zero T
			return zero, false
		}

		if l.head.CompareAndSwap(head, next) {
			l.length.Add(-1)
			return head.value, true
		}
	}
}

func (l *LockFreeList[T]) Contains(value T) bool {
	current := l.head.Load()
	for current != nil {
		if any(current.value) == any(value) {
			return true
		}
		current = current.next.Load()
	}
	return false
}

func (l *LockFreeList[T]) Length() int64 {
	return l.length.Load()
}

func (l *LockFreeList[T]) IsEmpty() bool {
	return l.head.Load().next.Load() == nil
}

type LockFreeSet[T comparable] struct {
	lockFreeList *LockFreeList[T]
}

func NewLockFreeSet[T comparable]() *LockFreeSet[T] {
	return &LockFreeSet[T]{
		lockFreeList: NewLockFreeList[T](),
	}
}

func (s *LockFreeSet[T]) Add(value T) bool {
	if s.Contains(value) {
		return false
	}
	s.lockFreeList.Insert(value)
	return true
}

func (s *LockFreeSet[T]) Remove(value T) bool {
	return s.lockFreeList.Delete(value)
}

func (s *LockFreeSet[T]) Contains(value T) bool {
	return s.lockFreeList.Contains(value)
}

func (s *LockFreeSet[T]) Size() int64 {
	return s.lockFreeList.Length()
}

func (l *LockFreeList[T]) Delete(value T) bool {
	current := l.head.Load()
	for current != nil {
		next := current.next.Load()
		if next != nil && any(next.value) == any(value) {
			if current.next.CompareAndSwap(next, next.next.Load()) {
				l.length.Add(-1)
				return true
			}
			current = l.head.Load()
			continue
		}
		current = next
	}
	return false
}

type LockFreeRingBuffer[T any] struct {
	buffer   []T
	capacity int
	head     atomic.Int64
	tail     atomic.Int64
	length   atomic.Int64
}

func NewLockFreeRingBuffer[T any](capacity int) *LockFreeRingBuffer[T] {
	return &LockFreeRingBuffer[T]{
		buffer:   make([]T, capacity),
		capacity: capacity,
	}
}

func (rb *LockFreeRingBuffer[T]) Push(value T) bool {
	cap := int64(rb.capacity)
	length := rb.length.Load()

	if length >= cap {
		return false
	}

	tail := rb.tail.Load()
	rb.buffer[tail] = value
	rb.tail.Store((tail + 1) % cap)
	rb.length.Add(1)

	return true
}

func (rb *LockFreeRingBuffer[T]) Pop() (T, bool) {
	if rb.length.Load() == 0 {
		var zero T
		return zero, false
	}

	head := rb.head.Load()
	value := rb.buffer[head]
	rb.head.Store((head + 1) % int64(rb.capacity))
	rb.length.Add(-1)

	return value, true
}

func (rb *LockFreeRingBuffer[T]) IsEmpty() bool {
	return rb.length.Load() == 0
}

func (rb *LockFreeRingBuffer[T]) IsFull() bool {
	return rb.length.Load() >= int64(rb.capacity)
}

func (rb *LockFreeRingBuffer[T]) Size() int64 {
	return rb.length.Load()
}

func (rb *LockFreeRingBuffer[T]) Capacity() int {
	return rb.capacity
}

type LockFreeCounter struct {
	value    atomic.Int64
	minValue atomic.Int64
	maxValue atomic.Int64
}

func NewLockFreeCounter() *LockFreeCounter {
	return &LockFreeCounter{}
}

func (c *LockFreeCounter) Increment(delta int64) int64 {
	newVal := c.value.Add(delta)

	for {
		minVal := c.minValue.Load()
		if newVal >= minVal || c.minValue.CompareAndSwap(minVal, newVal) {
			break
		}
	}

	for {
		maxVal := c.maxValue.Load()
		if newVal <= maxVal || c.maxValue.CompareAndSwap(maxVal, newVal) {
			break
		}
	}

	return newVal
}

func (c *LockFreeCounter) Decrement(delta int64) int64 {
	return c.Increment(-delta)
}

func (c *LockFreeCounter) Get() int64 {
	return c.value.Load()
}

func (c *LockFreeCounter) GetMin() int64 {
	return c.minValue.Load()
}

func (c *LockFreeCounter) GetMax() int64 {
	return c.maxValue.Load()
}

func (c *LockFreeCounter) Reset() {
	c.value.Store(0)
	c.minValue.Store(0)
	c.maxValue.Store(0)
}

type LockFreeBitmap struct {
	bits    []uint64
	size    int
	numWords int
}

func NewLockFreeBitmap(size int) *LockFreeBitmap {
	numWords := (size + 63) / 64
	return &LockFreeBitmap{
		bits:     make([]uint64, numWords),
		size:     size,
		numWords: numWords,
	}
}

func (b *LockFreeBitmap) Set(bit int) bool {
	if bit < 0 || bit >= b.size {
		return false
	}

	word := bit / 64
	offset := bit % 64
	oldVal := atomic.LoadUint64(&b.bits[word])
	mask := uint64(1) << offset
	newVal := oldVal | mask

	return atomic.CompareAndSwapUint64(&b.bits[word], oldVal, newVal)
}

func (b *LockFreeBitmap) Clear(bit int) bool {
	if bit < 0 || bit >= b.size {
		return false
	}

	word := bit / 64
	offset := bit % 64
	oldVal := atomic.LoadUint64(&b.bits[word])
	mask := ^(uint64(1) << offset)
	newVal := oldVal & mask

	return atomic.CompareAndSwapUint64(&b.bits[word], oldVal, newVal)
}

func (b *LockFreeBitmap) Test(bit int) bool {
	if bit < 0 || bit >= b.size {
		return false
	}

	word := bit / 64
	offset := bit % 64
	return (atomic.LoadUint64(&b.bits[word]) & (uint64(1) << offset)) != 0
}

func (b *LockFreeBitmap) FindFirstClear() int {
	for i := 0; i < b.numWords; i++ {
		word := atomic.LoadUint64(&b.bits[i])
		if word != ^uint64(0) {
			for j := 0; j < 64; j++ {
				if (word & (uint64(1) << j)) == 0 {
					bit := i*64 + j
					if bit < b.size {
						return bit
					}
				}
			}
		}
	}
	return -1
}

func (b *LockFreeBitmap) FindFirstSet() int {
	for i := 0; i < b.numWords; i++ {
		word := atomic.LoadUint64(&b.bits[i])
		if word != 0 {
			for j := 0; j < 64; j++ {
				if (word & (uint64(1) << j)) != 0 {
					return i*64 + j
				}
			}
		}
	}
	return -1
}

func (b *LockFreeBitmap) Count() int {
	count := 0
	for i := 0; i < b.numWords; i++ {
		word := atomic.LoadUint64(&b.bits[i])
		count += popcount(word)
	}
	return count
}

func popcount(x uint64) int {
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

type LockFreeDeque[T any] struct {
	buffer   []T
	capacity int
	head     atomic.Int64
	tail     atomic.Int64
}

func NewLockFreeDeque[T any](capacity int) *LockFreeDeque[T] {
	return &LockFreeDeque[T]{
		buffer:   make([]T, capacity),
		capacity: capacity,
	}
}

func (d *LockFreeDeque[T]) PushLeft(value T) bool {
	cap := int64(d.capacity)
	head := d.head.Load() - 1
	if head < 0 {
		head = cap - 1
	}

	if head == d.tail.Load() {
		return false
	}

	d.buffer[head] = value
	d.head.Store(head)
	return true
}

func (d *LockFreeDeque[T]) PushRight(value T) bool {
	cap := int64(d.capacity)
	tail := d.tail.Load()

	if (tail+1)%cap == d.head.Load() {
		return false
	}

	d.buffer[tail] = value
	d.tail.Store((tail + 1) % cap)
	return true
}

func (d *LockFreeDeque[T]) PopLeft() (T, bool) {
	cap := int64(d.capacity)
	head := d.head.Load()

	if head == d.tail.Load() {
		var zero T
		return zero, false
	}

	value := d.buffer[head]
	d.head.Store((head + 1) % cap)
	return value, true
}

func (d *LockFreeDeque[T]) PopRight() (T, bool) {
	cap := int64(d.capacity)
	tail := d.tail.Load()

	if d.head.Load() == tail {
		var zero T
		return zero, false
	}

	tail = tail - 1
	if tail < 0 {
		tail = cap - 1
	}

	value := d.buffer[tail]
	d.tail.Store(tail)
	return value, true
}

func (d *LockFreeDeque[T]) IsEmpty() bool {
	return d.head.Load() == d.tail.Load()
}

func (d *LockFreeDeque[T]) IsFull() bool {
	cap := int64(d.capacity)
	return (d.tail.Load()+1)%cap == d.head.Load()
}

func (d *LockFreeDeque[T]) Size() int {
	cap := int64(d.capacity)
	head := d.head.Load()
	tail := d.tail.Load()

	if tail >= head {
		return int(tail - head)
	}
	return int(cap - head + tail)
}
