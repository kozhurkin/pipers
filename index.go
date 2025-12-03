package pipers

import "context"

func printDebug(template string, rest ...interface{}) {
	//fmt.Printf("pipers:  [ %v ]    "+template+"\n", append([]interface{}{time.Now().String()[0:25]}, rest...)...)
}

func FromFuncs[T any](funcs ...func() (T, error)) *FliperSolver[T] {
	ps := FliperSolver[T]{
		flipers: make(Flipers[T], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFunc(f)
	}
	return &ps
}

func FromFuncsCtx[T any](funcs ...func(context.Context) (T, error)) *FliperSolver[T] {
	ps := FliperSolver[T]{
		flipers: make(Flipers[T], 0, len(funcs)),
	}
	for _, f := range funcs {
		ps.AddFuncCtx(f)
	}
	return &ps
}

func FromArgs[T any, A any](args []A, f func(int, A) (T, error)) *FliperSolver[T] {
	funcs := make([]func() (T, error), len(args))
	for i, v := range args {
		i, v := i, v
		funcs[i] = func() (T, error) {
			return f(i, v)
		}
	}
	return FromFuncs(funcs...)
}

func FromArgsCtx[T any, A any](args []A, f func(context.Context, int, A) (T, error)) *FliperSolver[T] {
	funcs := make([]func(ctx context.Context) (T, error), len(args))
	for i, v := range args {
		i, v := i, v
		funcs[i] = func(ctx context.Context) (T, error) {
			return f(ctx, i, v)
		}
	}
	return FromFuncsCtx(funcs...)
}

func Ref[T any](p *T, f func() (T, error)) func() (interface{}, error) {
	return func() (interface{}, error) {
		res, err := f()
		*p = res
		return res, err
	}
}

func Map[K comparable, T any](keys []K, values []T) map[K]T {
	if len(keys) != len(values) {
		return nil
	}
	res := make(map[K]T, len(keys))
	for i, k := range keys {
		res[k] = values[i]
	}
	return res
}

func Flatten[T any](lists [][]T) []T {
	var res []T
	for _, list := range lists {
		res = append(res, list...)
	}
	return res
}
