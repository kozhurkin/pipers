package pipers

import (
	"context"
	"sync"

	"github.com/kozhurkin/pipers/flight"
)

type FliperSolver[T any] struct {
	flipers     Flipers[T]
	concurrency int
	context     context.Context
	mu          sync.Mutex
}

func (ps *FliperSolver[T]) getContext() context.Context {
	if ps.context == nil {
		return context.Background()
	}
	return ps.context
}

func (ps *FliperSolver[T]) Context(ctx context.Context) *FliperSolver[T] {
	ps.context = ctx
	return ps
}

func (ps *FliperSolver[T]) Concurrency(concurrency int) *FliperSolver[T] {
	ps.concurrency = concurrency
	return ps
}

func (ps *FliperSolver[T]) Add(p *flight.Flight[T]) *FliperSolver[T] {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.flipers = append(ps.flipers, p)
	return ps
}

func (ps *FliperSolver[T]) AddFunc(f func() (T, error)) *FliperSolver[T] {
	p := flight.NewFlight(f)
	return ps.Add(p)
}

func (ps *FliperSolver[T]) AddFuncCtx(f func(ctx context.Context) (T, error)) *FliperSolver[T] {
	p := flight.NewFlight(func() (T, error) {
		return f(ps.getContext())
	})
	return ps.Add(p)
}

func (ps *FliperSolver[T]) FirstError() error {
	ctx, cancel := context.WithCancel(ps.getContext())
	defer cancel()
	ps.flipers.Run(ctx, ps.concurrency, 1)
	return ps.flipers.FirstError(ctx)
}

func (ps *FliperSolver[T]) FirstNErrors(n int) Errors {
	ctx, cancel := context.WithCancel(ps.getContext())
	defer cancel()
	ps.flipers.Run(ctx, ps.concurrency, n)
	return ps.flipers.FirstNErrors(ctx, n)
}

func (ps *FliperSolver[T]) ErrorsAll() Errors {
	return ps.FirstNErrors(0)
}

func (ps *FliperSolver[T]) Results() Results[T] {
	return ps.flipers.Results()
}

func (ps *FliperSolver[T]) Resolve() ([]T, error) {
	err := ps.FirstError()
	return ps.Results(), err
}

func (ps *FliperSolver[T]) Tail() <-chan struct{} {
	return ps.flipers.Tail()
}
