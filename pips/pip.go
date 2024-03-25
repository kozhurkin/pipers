package pips

func NewPip[R any](f func() R) chan R {
	out := make(chan R, 1)
	go func() {
		out <- f()
		close(out)
	}()
	return out
}

func FromFuncs[R any](funcs ...func() R) []R {
	pips := make([]chan R, len(funcs))
	for i, f := range funcs {
		pips[i] = NewPip(f)
	}
	return FromPips(pips...)
}

func FromPips[R any](pips ...chan R) []R {
	res := make([]R, len(pips))
	for i, p := range pips {
		res[i] = <-p
	}
	return res
}

func FromArgs[R any, A any](args []A, f func(int, A) R) []R {
	pips := make([]chan R, len(args))
	for i, a := range args {
		i, a := i, a
		pips[i] = NewPip(func() R {
			return f(i, a)
		})
	}
	return FromPips(pips...)
}
