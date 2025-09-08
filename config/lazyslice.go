package config

import "sync"

// LazySlice is a lazily initialized slice
type LazySlice[T any] struct {
	once  sync.Once
	items []T
	fn    func() []T
}

// NewLazySlice creates a new LazySlice with the given initialization function
func NewLazySlice[T any](fn func() []T) *LazySlice[T] {
	return &LazySlice[T]{
		fn: fn,
	}
}

// Get returns the slice, initializing it if necessary
func (ls *LazySlice[T]) Get() []T {
	ls.once.Do(func() {
		if ls.fn != nil {
			ls.items = ls.fn()
		}
	})
	return ls.items
}