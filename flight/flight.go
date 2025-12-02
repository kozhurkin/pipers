package flight

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrCanceled возвращается, если Flight был отменён до запуска fn.
var ErrCanceled = errors.New("flight: canceled")

// Flight хранит состояние одного вычисления значения типа T.
// Позволяет дождаться результата через Wait и гарантирует единичный запуск fn.
type Flight[T any] struct {
	done chan struct{}
	res  T
	err  error
	fn   func() (T, error)
	once sync.Once

	hits     uint64
	started  uint64
	canceled uint64
}

// NewFlight создаёт новый Flight для выполнения функции fn.
func NewFlight[T any](fn func() (T, error)) *Flight[T] {
	return &Flight[T]{
		done: make(chan struct{}),
		fn:   fn,
	}
}

// Done возвращает канал, который закрывается по завершении fn.
func (f *Flight[T]) Done() <-chan struct{} {
	return f.done
}

// Wait блокируется до завершения fn и возвращает результат и ошибку.
func (f *Flight[T]) Wait() (T, error) {
	<-f.done
	atomic.AddUint64(&f.hits, 1)
	return f.res, f.err
}

// OnDone блокируется до завершения fn (до закрытия канала done),
// а затем синхронно выполняет переданную функцию fn, передавая
// ему результат res и ошибку err.
// Если нужна асинхронность, вызывающий код может использовать go f.OnDone(fn).
func (f *Flight[T]) OnDone(fn func(res T, err error)) {
	<-f.done
	fn(f.res, f.err)
}

// Hits возвращает количество обращений к результату f.res через Wait.
// Безопасно для конкурентного использования.
func (f *Flight[T]) Hits() int64 {
	return int64(atomic.LoadUint64(&f.hits))
}

// Started возвращает true, если выполнение fn было запущено (синхронно или асинхронно).
// Безопасно для конкурентного использования.
func (f *Flight[T]) Started() bool {
	return atomic.LoadUint64(&f.started) == 1
}

// Canceled возвращает true, если Flight был отменён до запуска fn.
// Безопасно для конкурентного использования.
func (f *Flight[T]) Canceled() bool {
	return atomic.LoadUint64(&f.canceled) == 1
}

// Cancel пытается отменить Flight до запуска fn.
// Если отмена удалась (fn ещё не запускался), возвращает true и устанавливает ошибку ErrCanceled.
// Если fn уже был запущен или завершён, возвращает false и ничего не меняет.
func (f *Flight[T]) Cancel() bool {
	canceled := false

	f.once.Do(func() {
		canceled = true
		atomic.StoreUint64(&f.canceled, 1)

		var zero T
		f.res = zero
		f.err = ErrCanceled
		close(f.done)
	})

	return canceled
}

// execute выполняет функцию fn и закрывает канал done.
func (f *Flight[T]) execute() {
	f.res, f.err = f.fn()
	close(f.done)
}

// run выполняет fn ровно один раз.
// Если async == true, fn запускается в отдельной горутине, иначе выполняется синхронно.
// Для первого вызова возвращает true, для последующих — false.
func (f *Flight[T]) run(async bool) bool {
	first := false
	f.once.Do(func() {
		first = true
		atomic.StoreUint64(&f.started, 1)
		if async {
			go f.execute()
		} else {
			f.execute()
		}
	})
	return first
}

// Run выполняет fn ровно один раз синхронно (в той же горутине).
// Для первого вызова возвращает true, для последующих — false.
func (f *Flight[T]) Run() bool {
	return f.run(false)
}

// RunAsync запускает fn в отдельной горутине ровно один раз (не ждёт завершения).
// Для первого вызова возвращает true, для последующих — false.
func (f *Flight[T]) RunAsync() bool {
	return f.run(true)
}

// Then создаёт новый Flight[T], который будет выполнять функцию fn после завершения текущего Flight.
// Функция fn получает результат текущего выполнения и возвращает новое значение типа T и ошибку.
// Если текущий Flight завершился с ошибкой, fn не вызывается, и новый Flight сразу возвращает эту ошибку.
// Исходный Flight автоматически запускается синхронно для начала выполнения цепочки.
func (f *Flight[T]) Then(fn func(T) (T, error)) *Flight[T] {
	return ThenAny(f, fn)
}

// ThenAny создаёт новый Flight[R] из Flight[T], выполняя fn после завершения f.
// Это свободная функция (а не метод), потому что в Go методы не могут иметь собственные параметров типа.
func ThenAny[T, R any](f *Flight[T], fn func(T) (R, error)) *Flight[R] {
	return NewFlight(func() (R, error) {
		f.Run()
		res, err := f.Wait()
		if err != nil {
			var zero R
			return zero, err
		}
		return fn(res)
	})
}

// Catch создаёт новый Flight[T], который обрабатывает ошибку из текущего Flight.
// Если текущий Flight завершился без ошибки, результат просто прокидывается дальше.
// Если произошла ошибка, вызывается handler, который может вернуть восстановленное значение или
// другую ошибку.
func (f *Flight[T]) Catch(handler func(error) (T, error)) *Flight[T] {
	return NewFlight(func() (T, error) {
		f.Run()
		res, err := f.Wait()
		if err != nil {
			return handler(err)
		}
		return res, nil
	})
}

// Handle создаёт новый Flight[T], который всегда вызывает fn с результатом и ошибкой
// исходного Flight и возвращает то, что вернёт fn.
// В отличие от Then и Catch, fn получает и res и err одновременно и сам решает,
// как их интерпретировать.
func (f *Flight[T]) Handle(fn func(res T, err error) (T, error)) *Flight[T] {
	return NewFlight(func() (T, error) {
		f.Run()
		res, err := f.Wait()
		return fn(res, err)
	})
}
