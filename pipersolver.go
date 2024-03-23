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

func (ps *PiperSolver[R]) getContext() context.Context {
	if ps.context != nil {
		return ps.context
	} else {
		return context.Background()
	}
}

func (ps *PiperSolver[R]) Context(ctx context.Context) *PiperSolver[R] {
	ps.context = ctx
	return ps
}

func (ps *PiperSolver[R]) Run(errlim int) *PiperSolver[R] {
	ps.once.Do(func() {
		ps.Pipers.Run(ps.getContext(), ps.concurrency, errlim)
	})
	return ps
}

func (ps *PiperSolver[R]) FirstError() error {
	ps.Run(1)
	return ps.Pipers.FirstError(ps.getContext())
}

func (ps *PiperSolver[R]) FirstNErrors(n int) Errors {
	ps.Run(n)
	return ps.Pipers.FirstNErrors(ps.getContext(), n)
}

func (ps *PiperSolver[R]) ErrorsAll() Errors {
	ps.Run(0)
	return ps.Pipers.ErrorsAll(ps.getContext())
}

func (ps *PiperSolver[R]) Results() Results[R] {
	return ps.Pipers.Results()
}

func (ps *PiperSolver[R]) Resolve() ([]R, error) {
	err := ps.FirstError()
	return ps.Results(), err
}

func (ps *PiperSolver[R]) Wait() ([]R, error) {
	return ps.Resolve()
}
