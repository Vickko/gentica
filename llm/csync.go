package llm

import "sync"

// Map is a simplified wrapper around sync.Map for type safety
type Map[K comparable, V any] struct {
	m sync.Map
}

// NewMap creates a new thread-safe map
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{}
}

// Get retrieves a value from the map
func (m *Map[K, V]) Get(key K) (V, bool) {
	val, ok := m.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return val.(V), true
}

// Set sets a value in the map
func (m *Map[K, V]) Set(key K, value V) {
	m.m.Store(key, value)
}

// Delete removes a key from the map
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Del is an alias for Delete
func (m *Map[K, V]) Del(key K) {
	m.Delete(key)
}

// Range iterates over the map
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}

// Take removes and returns a value from the map
func (m *Map[K, V]) Take(key K) (V, bool) {
	val, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	return val.(V), true
}

// Seq returns all key-value pairs as a map copy for iteration
func (m *Map[K, V]) Seq() map[K]V {
	result := make(map[K]V)
	m.m.Range(func(k, v any) bool {
		result[k.(K)] = v.(V)
		return true
	})
	return result
}

// LazySlice is a simplified lazy-initialized slice using sync.Once
type LazySlice[T any] struct {
	once sync.Once
	mu   sync.RWMutex
	data []T
	fn   func() []T
}

// NewLazySlice creates a new lazy slice
func NewLazySlice[T any](fn func() []T) *LazySlice[T] {
	return &LazySlice[T]{
		fn: fn,
	}
}

// Get returns the slice data, initializing if needed
func (l *LazySlice[T]) Get() []T {
	l.once.Do(func() {
		l.data = l.fn()
	})
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.data
}

// Reset clears the cached data
func (l *LazySlice[T]) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.once = sync.Once{}
	l.data = nil
}

// Seq returns all elements as an iterator function
func (l *LazySlice[T]) Seq() func(yield func(int, T) bool) {
	return func(yield func(int, T) bool) {
		data := l.Get()
		for i, v := range data {
			if !yield(i, v) {
				break
			}
		}
	}
}

// Slice is a thread-safe slice
type Slice[T any] struct {
	mu   sync.RWMutex
	data []T
}

// NewSlice creates a new thread-safe slice
func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{
		data: make([]T, 0),
	}
}

// Append adds an element to the slice
func (s *Slice[T]) Append(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = append(s.data, item)
}

// Get returns a copy of the slice
func (s *Slice[T]) Get() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]T, len(s.data))
	copy(result, s.data)
	return result
}

// Len returns the length of the slice
func (s *Slice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}