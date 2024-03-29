package pips

func NewPip[T any](f func() T) chan T {
	out := make(chan T, 1)
	go func() {
		out <- f()
		close(out)
	}()
	return out
}

func FromFuncs[T any](funcs ...func() T) []T {
	pips := make([]chan T, len(funcs))
	for i, f := range funcs {
		pips[i] = NewPip(f)
	}
	return FromPips(pips...)
}

func FromPips[T any](pips ...chan T) []T {
	res := make([]T, len(pips))
	for i, p := range pips {
		res[i] = <-p
	}
	return res
}

func FromArgs[T any, A any](args []A, f func(int, A) T) []T {
	pips := make([]chan T, len(args))
	for i, a := range args {
		i, a := i, a
		pips[i] = NewPip(func() T {
			return f(i, a)
		})
	}
	return FromPips(pips...)
}
