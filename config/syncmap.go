package config

import (
	"sync"
)

// SyncMap is a thread-safe map implementation
type SyncMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

// NewSyncMap creates a new SyncMap
func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		data: make(map[K]V),
	}
}

// NewSyncMapFrom creates a new SyncMap from an existing map
func NewSyncMapFrom[K comparable, V any](m map[K]V) *SyncMap[K, V] {
	sm := &SyncMap[K, V]{
		data: make(map[K]V, len(m)),
	}
	for k, v := range m {
		sm.data[k] = v
	}
	return sm
}

// Set sets a key-value pair
func (m *SyncMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[K]V)
	}
	m.data[key] = value
}

// Get retrieves a value by key
func (m *SyncMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

// Del deletes a key
func (m *SyncMap[K, V]) Del(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// Take retrieves and removes a value by key
func (m *SyncMap[K, V]) Take(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if ok {
		delete(m.data, key)
	}
	return val, ok
}

// Len returns the number of items
func (m *SyncMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Range iterates over all key-value pairs
func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.data {
		if !f(k, v) {
			break
		}
	}
}

// Seq returns an iterator for range loops
func (m *SyncMap[K, V]) Seq() func(func(V) bool) {
	return func(yield func(V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for _, v := range m.data {
			if !yield(v) {
				return
			}
		}
	}
}

// Seq2 returns an iterator for range loops over key-value pairs
func (m *SyncMap[K, V]) Seq2() func(func(K, V) bool) {
	return func(yield func(K, V) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for k, v := range m.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Clone creates a deep copy of the map
func (m *SyncMap[K, V]) Clone() *SyncMap[K, V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newMap := &SyncMap[K, V]{
		data: make(map[K]V, len(m.data)),
	}
	for k, v := range m.data {
		newMap.data[k] = v
	}
	return newMap
}