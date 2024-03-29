package pipers

import "context"

func printDebug(template string, rest ...interface{}) {
	//fmt.Printf("pipers:  [ %v ]    "+template+"\n", append([]interface{}{time.Now().String()[0:25]}, rest...)...)
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

func FromFuncsCtx[R any](funcs ...func(context.Context) (R, error)) *PiperSolver[R] {
	ps := PiperSolver[R]{
		Pipers: make(Pipers[R], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFuncCtx(f)
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

func FromArgsCtx[R any, A any](args []A, f func(context.Context, int, A) (R, error)) *PiperSolver[R] {
	ps := PiperSolver[R]{
		Pipers: make(Pipers[R], 0, len(args)),
	}
	for i, v := range args {
		i, v := i, v
		ps.AddFuncCtx(func(ctx context.Context) (R, error) {
			return f(ctx, i, v)
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
