package pipers

import (
	"context"
	"sync"
	"sync/atomic"
)

type Pipers[T any] []Piper[T]

func (pp Pipers[T]) Run(ctx PipersContext, concurrency int) Pipers[T] {
	if concurrency == 0 || concurrency >= len(pp) {
		for _, p := range pp {
			p.Run()
		}
		return pp
	}
	go func() {
		traffic := make(chan struct{}, concurrency)
		defer func() {
			printDebug("close(traffic)")
			close(traffic)
		}()
		var errcnt int32
		limit := int32(ctx.Limit)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for _, p := range pp {
			p := p
			select {
			case <-ctx.Done():
				p.Close()
			case traffic <- struct{}{}:
				go func() {
					err := <-p.run()
					if err != nil && atomic.AddInt32(&errcnt, 1) >= limit {
						cancel()
					} else {
						<-traffic
					}
				}()
			}
		}
	}()
	return pp
}

func (pp Pipers[T]) ErrorsChan() chan error {
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
		printDebug("close(errchan)")
		close(errchan)
	}()

	return errchan
}

func (pp Pipers[T]) FirstNErrors(ctx PipersContext) Errors {
	errchan := pp.ErrorsChan()
	errs := make(Errors, 0, ctx.Limit)
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
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			return errs
		}
		if ctx.Limit > 0 && len(errs) == ctx.Limit {
			return errs
		}
	}
}

func (pp Pipers[T]) FirstError(ctx context.Context) error {
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

func (pp Pipers[T]) Results() Results[T] {
	res := make([]T, len(pp))
	for i, p := range pp {
		select {
		case res[i] = <-p.Val:
		default:
			continue
		}
	}
	return res
}
