// Package pool provides a generic object pool for types with Reset() method.
package pool

import "sync"

// Resetter is a constraint for types that have a Reset() method.
type Resetter interface {
	Reset()
}

// Pool is a generic object pool that stores objects of a specific type.
// The pool automatically calls Reset() on objects when they are returned,
// ensuring clean state for reuse.
type Pool[T Resetter] struct {
	pool sync.Pool
	new  func() T
}

// New creates and returns a pointer to a new Pool instance.
// The factory function is used to create new instances when the pool is empty.
func New[T Resetter](factory func() T) *Pool[T] {
	return &Pool[T]{
		new: factory,
	}
}

// Get returns an object from the pool, or creates a new one if the pool is empty.
// If an object is retrieved from the pool, it has been reset to a clean state.
func (p *Pool[T]) Get() T {
	obj := p.pool.Get()
	if obj != nil {
		return obj.(T)
	}
	return p.new()
}

// Put returns an object to the pool after resetting its state.
// The object's Reset() method is called to clear its state before returning it to the pool.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
