package pipers

import "context"

func printDebug(template string, rest ...interface{}) {
	//fmt.Printf("pipers:  [ %v ]    "+template+"\n", append([]interface{}{time.Now().String()[0:25]}, rest...)...)
}

func NewPiper[T any](f func() (T, error)) Piper[T] {
	return Piper[T]{
		Out: make(chan T, 1),
		Err: make(chan error, 1),
		Job: f,
	}
}

func FromFuncs[T any](funcs ...func() (T, error)) *PiperSolver[T] {
	ps := PiperSolver[T]{
		Pipers: make(Pipers[T], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFunc(f)
	}
	return &ps
}

func FromFuncsCtx[T any](funcs ...func(context.Context) (T, error)) *PiperSolver[T] {
	ps := PiperSolver[T]{
		Pipers: make(Pipers[T], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFuncCtx(f)
	}
	return &ps
}

func FromArgs[T any, A any](args []A, f func(int, A) (T, error)) *PiperSolver[T] {
	ps := PiperSolver[T]{
		Pipers: make(Pipers[T], 0, len(args)),
	}
	for i, v := range args {
		i, v := i, v
		ps.AddFunc(func() (T, error) {
			return f(i, v)
		})
	}
	return &ps
}

func FromArgsCtx[T any, A any](args []A, f func(context.Context, int, A) (T, error)) *PiperSolver[T] {
	ps := PiperSolver[T]{
		Pipers: make(Pipers[T], 0, len(args)),
	}
	for i, v := range args {
		i, v := i, v
		ps.AddFuncCtx(func(ctx context.Context) (T, error) {
			return f(ctx, i, v)
		})
	}
	return &ps
}

func Ref[R any](p *R, f func() (R, error)) func() (interface{}, error) {
	return func() (interface{}, error) {
		res, err := f()
		*p = res
		return res, err
	}
}
