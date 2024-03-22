package pipers

import (
	"fmt"
	"time"
)

func printDebug(template string, rest ...interface{}) {
	var debug bool
	//debug = true
	if debug {
		args := append([]interface{}{time.Now().String()[0:25]}, rest...)
		fmt.Printf("pipers:  [ %v ]    "+template+"\n", args...)
	}
}

func NewPiper[R any](f func() (R, error)) Piper[R] {
	return Piper[R]{
		Out: make(chan R, 1),
		Err: make(chan error, 1),
		Job: f,
	}
}

func FromFuncs[R any](funcs ...func() (R, error)) *PiperSolver[R] {
	ps := PiperSolver[R]{
		Pipers: make(Pipers[R], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFunc(f)
	}
	return &ps
}

func FromArgs[R any, A any](args []A, f func(int, A) (R, error)) *PiperSolver[R] {
	ps := PiperSolver[R]{
		Pipers: make(Pipers[R], 0, len(args)),
	}
	for i, v := range args {
		i, v := i, v
		ps.AddFunc(func() (R, error) {
			return f(i, v)
		})
	}
	return &ps
}

func Ref[P any](p *P, f func() (P, error)) func() (interface{}, error) {
	return func() (interface{}, error) {
		res, err := f()
		*p = res
		return res, err
	}
}
