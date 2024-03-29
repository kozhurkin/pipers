package pipers

import (
	"context"
	"sync"
)

type PiperSolver[T any] struct {
	Pipers[T]
	concurrency int
	context     context.Context
	once        sync.Once
	mu          sync.RWMutex
}

func (ps *PiperSolver[T]) Add(p Piper[T]) *PiperSolver[T] {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Pipers = append(ps.Pipers, p)
	return ps
}

func (ps *PiperSolver[T]) AddFunc(f func() (T, error)) *PiperSolver[T] {
	p := NewPiper(f)
	return ps.Add(p)
}

func (ps *PiperSolver[T]) AddFuncCtx(f func(ctx context.Context) (T, error)) *PiperSolver[T] {
	p := NewPiper(func() (T, error) {
		return f(ps.Ctx())
	})
	return ps.Add(p)
}

func (ps *PiperSolver[T]) Concurrency(concurrency int) *PiperSolver[T] {
	ps.concurrency = concurrency
	return ps
}

func (ps *PiperSolver[T]) Run(ctx PipersContext) {
	ps.once.Do(func() {
		ps.Pipers.Run(ctx, ps.concurrency)
	})
}

func (ps *PiperSolver[T]) createContextWithCancelAndLimit(n int) (PipersContext, context.CancelFunc) {
	var cancel context.CancelFunc
	ctx := ps.context
	if ps.context == nil {
		ctx = context.Background()
	}
	ctx, cancel = context.WithCancel(ctx)
	ps.setContext(ctx)
	return PipersContext{ctx, n}, cancel
}

func (ps *PiperSolver[T]) setContext(ctx context.Context) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.context = ctx
}

func (ps *PiperSolver[T]) Ctx() context.Context {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.context
}

func (ps *PiperSolver[T]) Context(ctx context.Context) *PiperSolver[T] {
	ps.setContext(ctx)
	return ps
}

func (ps *PiperSolver[T]) FirstError() error {
	ctx, cancel := ps.createContextWithCancelAndLimit(1)
	defer cancel()
	ps.Run(ctx)
	return ps.Pipers.FirstError(ctx)
}

func (ps *PiperSolver[T]) FirstNErrors(n int) Errors {
	ctx, cancel := ps.createContextWithCancelAndLimit(n)
	defer cancel()
	ps.Run(ctx)
	return ps.Pipers.FirstNErrors(ctx)
}

func (ps *PiperSolver[T]) ErrorsAll() Errors {
	return ps.FirstNErrors(0)
}

func (ps *PiperSolver[T]) Results() Results[T] {
	return ps.Pipers.Results()
}

func (ps *PiperSolver[T]) Resolve() ([]T, error) {
	err := ps.FirstError()
	return ps.Results(), err
}

func (ps *PiperSolver[T]) Wait() ([]T, error) {
	return ps.Resolve()
}
