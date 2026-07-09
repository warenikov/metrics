// Package pool provides a generic, type-safe wrapper around [sync.Pool].
//
// Objects stored in the pool must implement the [Resetter] interface so that
// stale state is never leaked between callers: [Pool.Put] calls Reset()
// automatically before returning the object to the pool.
//
// Typical usage — create the pool once at package level and reuse it:
//
//	var bufPool = pool.New(func() *bytes.Buffer { return &bytes.Buffer{} })
//
//	func process(data []byte) {
//	    buf := bufPool.Get()
//	    defer bufPool.Put(buf)
//	    buf.Write(data)
//	    // ... use buf ...
//	}
//
// The type parameter T is almost always a pointer type (e.g. *MyStruct)
// because Reset() is conventionally implemented on pointer receivers.
package pool

import "sync"

// Resetter must be implemented by every type stored in [Pool].
// Reset brings the object back to its zero state so it can be reused safely.
type Resetter interface {
	Reset()
}

// Pool is a type-safe, generic wrapper around [sync.Pool].
// T must implement [Resetter]; Put calls Reset before returning the object.
type Pool[T Resetter] struct {
	p sync.Pool
}

// New creates a Pool whose factory is newFn. newFn is called by Get when the
// pool is empty. newFn may be nil, in which case Get returns the zero value
// of T instead of allocating.
func New[T Resetter](newFn func() T) *Pool[T] {
	p := &Pool[T]{}
	if newFn != nil {
		p.p.New = func() any { return newFn() }
	}
	return p
}

// Get retrieves an object from the pool, allocating a new one via the factory
// if the pool is currently empty. If no factory was given to New, Get returns
// the zero value of T instead.
func (p *Pool[T]) Get() T {
	v := p.p.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T) //nolint:forcetypeassert
}

// Put resets v and returns it to the pool for future reuse.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
