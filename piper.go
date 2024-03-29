package pipers

type Piper[T any] struct {
	Out chan T
	Err chan error
	Job func() (T, error)
}

func (p Piper[T]) Close() Piper[T] {
	printDebug("% v.Close()", p)
	close(p.Out)
	close(p.Err)
	return p
}
func (p Piper[T]) Run() Piper[T] {
	p.run()
	return p
}
func (p Piper[T]) run() chan error {
	printDebug("% v.run()  ", p)
	done := make(chan error, 1)
	go func() {
		v, e := p.Job()
		if e != nil {
			done <- e
		}
		close(done)
		p.Out <- v
		p.Err <- e
		p.Close()
	}()
	return done
}
