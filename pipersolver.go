package pipers

import (
	"context"
	"sync"
)

type PiperSolver[R any] struct {
	Pipers[R]
	concurrency int
	context     context.Context
	once        sync.Once
	mu          sync.RWMutex
}

func (ps *PiperSolver[R]) Add(p Piper[R]) *PiperSolver[R] {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	ps.Pipers = append(ps.Pipers, p)
	return ps
}

func (ps *PiperSolver[R]) AddFunc(f func() (R, error)) *PiperSolver[R] {
	p := NewPiper(f)
	return ps.Add(p)
}

func (ps *PiperSolver[R]) Concurrency(concurrency int) *PiperSolver[R] {
	ps.concurrency = concurrency
	return ps
}

func (ps *PiperSolver[R]) Run(ctx PipersContext) {
	ps.once.Do(func() {
		ps.Pipers.Run(ctx, ps.concurrency)
	})
}

func (ps *PiperSolver[R]) createContextWithCancelAndLimit(n int) (PipersContext, context.CancelFunc) {
	var cancel context.CancelFunc
	ctx := ps.context
	if ps.context == nil {
		ctx = context.Background()
	}
	ctx, cancel = context.WithCancel(ctx)
	return PipersContext{ctx, n}, cancel
}

func (ps *PiperSolver[R]) Context(ctx context.Context) *PiperSolver[R] {
	ps.context = ctx
	return ps
}

func (ps *PiperSolver[R]) FirstError() error {
	ctx, cancel := ps.createContextWithCancelAndLimit(1)
	defer cancel()
	ps.Run(ctx)
	return ps.Pipers.FirstError(ctx)
}

func (ps *PiperSolver[R]) FirstNErrors(n int) Errors {
	ctx, cancel := ps.createContextWithCancelAndLimit(n)
	defer cancel()
	ps.Run(ctx)
	return ps.Pipers.FirstNErrors(ctx)
}

func (ps *PiperSolver[R]) ErrorsAll() Errors {
	return ps.FirstNErrors(0)
}

func (ps *PiperSolver[R]) Results() Results[R] {
	return ps.Pipers.Results()
}

func (ps *PiperSolver[R]) Resolve() ([]R, error) {
	err := ps.FirstError()
	return ps.Results(), err
}
