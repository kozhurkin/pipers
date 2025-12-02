package pipers

import (
	"context"
	"sync"

	"github.com/kozhurkin/pipers/flight"
)

// Flipers представляет собой набор указателей на Flight[T],
// над которыми выполняются групповые операции.
type Flipers[T any] []*flight.Flight[T]

// Run запускает вычисления для всех Flight из набора с учётом ограничения
// по concurrency (количество одновременно работающих задач).
// Если concurrency == 0 или больше числа задач, все задачи запускаются
// параллельно без ограничения.
// Остановка дальнейших запусков контролируется только переданным контекстом.
func (pp Flipers[T]) Run(ctx context.Context, concurrency int) Flipers[T] {
	if concurrency == 0 || concurrency >= len(pp) {
		for _, p := range pp {
			go p.Run()
		}
		return pp
	}
	go func() {
		traffic := make(chan struct{}, concurrency)
		defer func() {
			printDebug("close(traffic)")
			close(traffic)
		}()

		for _, p := range pp {
			p := p
			select {
			case <-ctx.Done():
				return // context canceled
			case traffic <- struct{}{}:
				go func() {
					defer func() {
						<-traffic
					}()
					select {
					case <-ctx.Done():
						// контекст уже отменён — не запускаем задачу
						return
					default:
						p.Run()
					}
				}()
			}
		}
	}()
	return pp
}

// ErrorsChan возвращает буферизированный канал ошибок, в который будут
// отправляться ошибки завершившихся Flight. Канал закрывается после
// завершения всех полётов или при завершении контекста.
func (pp Flipers[T]) ErrorsChan(ctx context.Context) chan error {
	errchan := make(chan error, len(pp))
	wg := sync.WaitGroup{}

	for _, p := range pp {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				if !p.Started() {
					return
				}
			case <-p.Done():
				// already done
			}
			if _, err := p.Wait(); err != nil {
				errchan <- err
			}
		}()
	}
	go func() {
		wg.Wait()
		printDebug("close(errchan)")
		close(errchan)
	}()

	return errchan
}

// Tail возвращает канал, который будет закрыт после завершения всех Flight,
// которые к моменту вызова уже были запущены (p.Started() == true).
// Предполагается, что используется после FirstError/FirstNErrors/ErrorsAll,
// когда запуск новых Flight уже прекращён.
func (pp Flipers[T]) Tail() <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		for _, p := range pp {
			if p.Started() {
				<-p.Done()
			}
		}
		close(ch)
	}()

	return ch
}

// FirstNErrors возвращает до limit первых ошибок из Flipers.
// Если limit <= 0, собираются все ошибки. В случае завершения контекста
// в результирующий срез также добавляется ошибка контекста.
func (pp Flipers[T]) FirstNErrors(ctx context.Context, limit int) Errors {
	errchan := pp.ErrorsChan(ctx)
	errs := make(Errors, 0, limit)
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
		if limit > 0 && limit == len(errs) {
			return errs
		}
	}
}

// FirstError возвращает первую ошибку из Flipers либо nil, если ошибок не было.
// Если контекст завершён раньше, возвращается ошибка контекста.
func (pp Flipers[T]) FirstError(ctx context.Context) error {
	errchan := pp.ErrorsChan(ctx)
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

// Results возвращает срез результатов завершившихся на данный момент Flight.
// Для ещё не завершившихся полётов в срезе остаётся zero-value соответствующего типа.
func (pp Flipers[T]) Results() Results[T] {
	res := make([]T, len(pp))
	for i, p := range pp {
		select {
		case <-p.Done():
			res[i], _ = p.Wait()
		default:
			continue
		}
	}
	return res
}
