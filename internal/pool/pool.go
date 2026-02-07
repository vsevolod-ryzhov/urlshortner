package pool

import (
	"sync"
)

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	pool    sync.Pool
	newFunc func() T
}

func New[T Resetter](newFunc func() T) *Pool[T] {
	p := &Pool[T]{
		newFunc: newFunc,
	}

	p.pool.New = func() interface{} {
		obj := newFunc()
		obj.Reset()
		return obj
	}

	return p
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}
