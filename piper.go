package pipers

type Piper[T any] struct {
	Val chan T
	Err chan error
	Job func() (T, error)
}

func NewPiper[T any](f func() (T, error)) Piper[T] {
	return Piper[T]{
		Val: make(chan T, 1),
		Err: make(chan error, 1),
		Job: f,
	}
}

func (p Piper[T]) Close() Piper[T] {
	printDebug("% v.Close()", p)
	close(p.Val)
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
		p.Val <- v
		p.Err <- e
		p.Close()
	}()
	return done
}
