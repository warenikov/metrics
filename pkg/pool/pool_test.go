package pool_test

import (
	"sync"
	"testing"

	"metrics/pkg/pool"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// item is a test value whose Reset() call count is tracked.
type item struct {
	name       string
	tags       []string
	value      int
	resetCalls int
}

func (it *item) Reset() {
	it.value = 0
	it.name = ""
	it.tags = it.tags[:0]
	it.resetCalls++
}

func newItem() *item {
	return &item{tags: make([]string, 0, 4)}
}

func TestNew(t *testing.T) {
	p := pool.New(newItem)
	require.NotNil(t, p)
}

func TestGet_ReturnsObject(t *testing.T) {
	p := pool.New(newItem)
	it := p.Get()
	require.NotNil(t, it)
}

func TestNew_NilFactory_GetReturnsZeroValue(t *testing.T) {
	p := pool.New[*item](nil)
	it := p.Get()
	assert.Nil(t, it, "Get without a factory must return the zero value of T")
}

func TestGet_UsesFactory(t *testing.T) {
	calls := 0
	p := pool.New(func() *item {
		calls++
		return newItem()
	})

	_ = p.Get()
	_ = p.Get()
	// sync.Pool may or may not cache across GC; at least one factory call expected.
	assert.GreaterOrEqual(t, calls, 1)
}

func TestPut_CallsReset(t *testing.T) {
	p := pool.New(newItem)
	it := p.Get()
	it.value = 42
	it.name = "hello"
	it.tags = append(it.tags, "a", "b")

	p.Put(it)

	assert.Equal(t, 1, it.resetCalls, "Reset must be called on Put")
	assert.Equal(t, 0, it.value)
	assert.Equal(t, "", it.name)
	assert.Empty(t, it.tags)
}

func TestGetAfterPut_ReturnsResetObject(t *testing.T) {
	p := pool.New(newItem)

	it := p.Get()
	it.value = 99
	p.Put(it)

	// Retrieve from pool; may or may not be the same pointer (GC can discard),
	// but if it is the same it must be in reset state.
	it2 := p.Get()
	if it2 == it {
		assert.Equal(t, 0, it2.value, "recycled object must be in reset state")
	}
}

func TestPool_Concurrent(t *testing.T) {
	p := pool.New(newItem)
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			it := p.Get()
			it.value = 1
			it.name = "concurrent"
			it.tags = append(it.tags, "x")
			p.Put(it)
		}()
	}
	wg.Wait()
}

func TestPool_MultipleGetPutCycles(t *testing.T) {
	p := pool.New(newItem)

	for range 10 {
		it := p.Get()
		it.value = 7
		it.name = "cycle"
		p.Put(it)
		assert.GreaterOrEqual(t, it.resetCalls, 1)
	}
}
