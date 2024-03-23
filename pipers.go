package pipers

import (
	"context"
	"sync"
	"sync/atomic"
)

type Pipers[R any] []Piper[R]

func (pp Pipers[R]) runAtOnce() Pipers[R] {
	for _, p := range pp {
		p.Run()
	}
	return pp
}

func (pp Pipers[R]) Run(ctx context.Context, n int, errlim int) Pipers[R] {
	if n == 0 || n == len(pp) {
		return pp.runAtOnce()
	}
	go func() {
		traffic := make(chan struct{}, n)
		catch := make(chan struct{})
		var once sync.Once
		var cnt int32
		defer func() {
			printDebug("close(traffic)")
			close(traffic)
			once.Do(func() {
				printDebug("close(catch)")
				close(catch)
			})
		}()
		for _, p := range pp {
			p := p
			select {
			case <-ctx.Done():
				p.Close()
			case <-catch:
				p.Close()
			case traffic <- struct{}{}:
				go func() {
					err := <-p.run()
					if err != nil && atomic.AddInt32(&cnt, 1) >= int32(errlim) {
						once.Do(func() {
							close(catch)
						})
					} else {
						<-traffic
					}
				}()
			}
		}
	}()
	return pp
}

func (pp Pipers[R]) Results() Results[R] {
	res := make([]R, len(pp))
	for i, p := range pp {
		select {
		case res[i] = <-p.Out:
		default:
			continue
		}
	}
	return res
}

func (pp Pipers[R]) ErrorsAll(ctx context.Context) Errors {
	return pp.FirstNErrors(ctx, 0)
}

func (pp Pipers[R]) FirstNErrors(ctx context.Context, n int) Errors {
	errs := make(Errors, 0, n)
	errchan := pp.ErrorsChan()
	done := make(chan struct{})
	var doneclosed int32
	go func() {
		if sig := <-ctx.Done(); atomic.LoadInt32(&doneclosed) == 0 {
			done <- sig
		}
	}()
	defer func() {
		atomic.AddInt32(&doneclosed, 1)
		close(done)
	}()
	for {
		select {
		case err, ok := <-errchan:
			if !ok {
				if len(errs) == 0 {
					return nil
				}
				return errs
			}
			errs = append(errs, err)
		case <-done:
			errs = append(errs, ctx.Err())
			return errs
		}
		if n > 0 && len(errs) == n {
			return errs
		}
	}
}

func (pp Pipers[R]) FirstError(ctx context.Context) error {
	errchan := pp.ErrorsChan()
	select {
	case err, ok := <-errchan:
		if !ok {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pp Pipers[R]) ErrorsChan() chan error {
	errchan := make(chan error, len(pp))
	wg := sync.WaitGroup{}

	wg.Add(len(pp))
	for _, p := range pp {
		p := p
		go func() {
			if err := <-p.Err; err != nil {
				errchan <- err
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		printDebug("************** close(errchan)")
		close(errchan)
	}()

	return errchan
}

func (pp Pipers[R]) Resolve(ctx context.Context) ([]R, error) {
	err := pp.FirstError(ctx)
	return pp.Results(), err
}
