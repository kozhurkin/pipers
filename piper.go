package pipers

type Piper[R any] struct {
	Out chan R
	Err chan error
	Job func() (R, error)
}

func (p Piper[R]) Close() Piper[R] {
	printDebug("% v.Close()", p)
	close(p.Out)
	close(p.Err)
	return p
}
func (p Piper[R]) Run() Piper[R] {
	p.run()
	return p
}
func (p Piper[R]) run() chan error {
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
