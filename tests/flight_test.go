package tests

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kozhurkin/pipers/flight"
)

func TestFlight_RunOnce(t *testing.T) {
	var calls int32

	f := flight.NewFlight(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(5 * time.Millisecond)
		return 42, nil
	})

	// Последовательные вызовы: первый должен вернуть true, последующие — false.
	ok := f.Run()
	require.True(t, ok, "first Run() should return true")

	ok = f.Run()
	require.False(t, ok, "second Run() should return false")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			f.Run()
		}(i)
	}
	wg.Wait()

	got := atomic.LoadInt32(&calls)
	require.Equal(t, int32(1), got, "fn should be called exactly once")
}

func TestFlight_WaitAndHits(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		return 42, nil
	})

	f.Run()

	v1, err1 := f.Wait()
	v2, err2 := f.Wait()

	require.NoError(t, err1, "first Wait() should not return error")
	require.NoError(t, err2, "second Wait() should not return error")
	require.Equal(t, 42, v1, "first Wait() should return 42")
	require.Equal(t, 42, v2, "second Wait() should return 42")
	require.Equal(t, int64(2), f.Hits(), "Hits() should equal number of Wait calls")
}

func TestFlight_DoneChannel(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		time.Sleep(5 * time.Millisecond)
		return 1, nil
	})

	// До запуска Run канал не должен быть закрыт.
	select {
	case <-f.Done():
		require.FailNow(t, "Done closed before Run")
	default:
	}

	done := f.Done()

	// Повторный неблокирующий select по сохранённому каналу Done:
	// до запуска Run он также не должен быть закрыт.
	select {
	case <-done:
		require.FailNow(t, "Done closed before Run on saved channel")
	default:
	}

	go f.Run()

	select {
	case <-done:
		// ok
	case <-time.After(10 * time.Millisecond):
		require.FailNow(t, "timeout waiting for Done to be closed")
	}
}

func TestFlight_Wait_ConcurrentHits(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		time.Sleep(5 * time.Millisecond)
		return 7, nil
	})

	f.RunAsync()

	// Ждём завершения вычисления.
	<-f.Done()

	const n = 10
	var wg sync.WaitGroup
	results := make([]int, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := f.Wait()
			require.NoError(t, err, "Wait() should not return error")
			results[i] = v
		}(i)
	}

	wg.Wait()

	for i, v := range results {
		require.Equalf(t, 7, v, "results[%d] should be 7", i)
	}
	require.Equal(t, int64(n), f.Hits(), "Hits() should equal number of Wait calls")
}

func TestFlight_Then_Success(t *testing.T) {
	base := flight.NewFlight(func() (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 2, nil
	})

	next := base.Then(func(v int) (int, error) {
		return v * 10, nil
	})

	next.Run()
	res, err := next.Wait()
	require.NoError(t, err, "Then() chain should not return error")
	require.Equal(t, 20, res, "Then() should transform 2 into 20")
}

func TestFlight_ThenAny_Success(t *testing.T) {
	base := flight.NewFlight(func() (int, error) {
		return 5, nil
	})

	next := flight.ThenAny(base, func(v int) (string, error) {
		return fmt.Sprintf("%d", v*2), nil
	})

	next.Run()
	res, err := next.Wait()
	require.NoError(t, err, "ThenAny() chain should not return error")
	require.Equal(t, "10", res, "ThenAny() should transform 5 into \"10\"")
}

func TestFlight_Then_ErrorPropagation(t *testing.T) {
	someErr := errors.New("boom")
	base := flight.NewFlight(func() (int, error) {
		return 0, someErr
	})

	var called int32
	next := base.Then(func(v int) (int, error) {
		// не должен вызываться
		atomic.AddInt32(&called, 1)
		return v + 1, nil
	})

	next.Run()
	_, err := next.Wait()
	require.ErrorIs(t, err, someErr, "Then() should propagate base error")
	require.Equal(t, int32(0), atomic.LoadInt32(&called), "next should not be called on error")
}

func TestFlight_ThenAny_ErrorPropagation(t *testing.T) {
	someErr := errors.New("boom")
	base := flight.NewFlight(func() (int, error) {
		return 0, someErr
	})

	var called int32
	next := flight.ThenAny(base, func(v int) (string, error) {
		atomic.AddInt32(&called, 1)
		return fmt.Sprintf("%d", v), nil
	})

	next.Run()
	_, err := next.Wait()
	require.ErrorIs(t, err, someErr, "ThenAny() should propagate base error")
	require.Equal(t, int32(0), atomic.LoadInt32(&called), "next should not be called on error")
}

func TestFlight_Catch(t *testing.T) {
	someErr := errors.New("boom")
	base := flight.NewFlight(func() (int, error) {
		return 0, someErr
	})

	recovered := base.Catch(func(err error) (int, error) {
		if !errors.Is(err, someErr) {
			return 0, err
		}
		return 123, nil
	})

	recovered.Run()
	res, err := recovered.Wait()
	require.NoError(t, err, "Catch() should recover from error")
	require.Equal(t, 123, res, "Catch() should return recovered value")
}

func TestFlight_Catch_NoErrorPassthrough(t *testing.T) {
	const value = 7

	base := flight.NewFlight(func() (int, error) {
		return value, nil
	})

	var called int32
	recovered := base.Catch(func(err error) (int, error) {
		atomic.AddInt32(&called, 1)
		return 0, fmt.Errorf("unexpected handler call: %v", err)
	})

	recovered.Run()
	res, err := recovered.Wait()
	require.NoError(t, err, "Catch() should pass through successful result")
	require.Equal(t, value, res, "Catch() should keep original value when there is no error")
	require.Equal(t, int32(0), atomic.LoadInt32(&called), "handler must not be called on success")
}

func TestFlight_After_CalledAfterCompletion(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		time.Sleep(5 * time.Millisecond)
		return 1, nil
	})

	afterCalled := make(chan struct{})

	go f.OnDone(func(v int, err error) {
		require.NoError(t, err)
		require.Equal(t, 1, v)
		close(afterCalled)
	})
	go f.Run() // порядок запуска не важен: After ждёт Done

	select {
	case <-afterCalled:
		// ok
	case <-time.After(20 * time.Millisecond):
		require.FailNow(t, "After callback was not called after Run completion")
	}
}

func TestFlight_After_AlreadyDone(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		return 2, nil
	})

	f.Run()
	<-f.Done()

	afterCalled := make(chan struct{})

	go f.OnDone(func(v int, err error) {
		require.NoError(t, err)
		require.Equal(t, 2, v)
		close(afterCalled)
	})

	select {
	case <-afterCalled:
		// ok: callback should fire immediately for already-completed Flight
	case <-time.After(5 * time.Millisecond):
		require.FailNow(t, "After callback was not called for already-completed Flight")
	}
}

func TestFlight_StartedFlag(t *testing.T) {
	f := flight.NewFlight(func() (int, error) {
		time.Sleep(5 * time.Millisecond)
		return 1, nil
	})

	require.False(t, f.Started(), "Started() should be false before Run")

	ok := f.Run()
	require.True(t, ok, "first Run() should return true")
	require.True(t, f.Started(), "Started() should be true after Run")

	// Дополнительные вызовы Run не должны менять Started().
	ok = f.Run()
	require.False(t, ok, "second Run() should return false")
	require.True(t, f.Started(), "Started() should remain true after repeated Run")
}

func TestFlight_RunAsyncOnce(t *testing.T) {
	var calls int32

	f := flight.NewFlight(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(5 * time.Millisecond)
		return 42, nil
	})

	const goroutines = 20
	var wg sync.WaitGroup
	var trueCount int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if f.RunAsync() {
				atomic.AddInt32(&trueCount, 1)
			}
		}()
	}

	wg.Wait()
	<-f.Done()

	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "fn should be called exactly once with RunAsync")
	require.Equal(t, int32(1), atomic.LoadInt32(&trueCount), "exactly one RunAsync call should return true")
	require.True(t, f.Started(), "Started() should be true after RunAsync")
}

func TestFlight_CancelBeforeRun(t *testing.T) {
	var calls int32

	f := flight.NewFlight(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		return 123, nil
	})

	require.False(t, f.Started(), "Started() should be false before Run or Cancel")
	require.False(t, f.Canceled(), "Canceled() should be false before Cancel")

	ok := f.Cancel()
	require.True(t, ok, "first Cancel() before Run should return true")
	require.True(t, f.Canceled(), "Canceled() should be true after successful Cancel")
	require.False(t, f.Started(), "Started() should remain false after Cancel before Run")

	ok = f.Cancel()
	require.False(t, ok, "second Cancel() should return false")

	// fn не должен был вызываться.
	require.Equal(t, int32(0), atomic.LoadInt32(&calls), "fn must not be called if Flight was canceled before Run")

	// Канал Done должен быть закрыт.
	select {
	case <-f.Done():
		// ok
	default:
		require.FailNow(t, "Done should be closed after Cancel")
	}

	v, err := f.Wait()
	require.ErrorIs(t, err, flight.ErrCanceled, "Wait() should return ErrCanceled after Cancel")
	require.Equal(t, 0, v, "Wait() should return zero value after Cancel")
}

func TestFlight_CancelAfterRun(t *testing.T) {
	var calls int32

	f := flight.NewFlight(func() (int, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(5 * time.Millisecond)
		return 42, nil
	})

	require.True(t, f.RunAsync(), "first RunAsync() should return true")

	// Ждём, пока запуск зафиксируется.
	require.Eventually(t, func() bool {
		return f.Started()
	}, 100*time.Millisecond, 1*time.Millisecond, "Started() should eventually become true after RunAsync")

	ok := f.Cancel()
	require.False(t, ok, "Cancel() after Run should return false")
	require.False(t, f.Canceled(), "Canceled() should remain false if Cancel called after Run")

	v, err := f.Wait()
	require.NoError(t, err, "Wait() should not return error when Cancel after Run")
	require.Equal(t, 42, v, "Wait() should return fn result when Cancel after Run")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "fn should still be called exactly once")
}

func TestFlight_Handle_Success(t *testing.T) {
	base := flight.NewFlight(func() (int, error) {
		return 10, nil
	})

	handled := base.Handle(func(res int, err error) (int, error) {
		require.NoError(t, err, "base error should be nil in Handle on success")
		require.Equal(t, 10, res, "Handle should receive original result")
		return res * 2, nil
	})

	handled.Run()
	v, err := handled.Wait()

	require.NoError(t, err, "Handle() chain should not return error on success")
	require.Equal(t, 20, v, "Handle() should be able to transform the result")
}

func TestFlight_Handle_ErrorRecovery(t *testing.T) {
	someErr := errors.New("boom")

	base := flight.NewFlight(func() (int, error) {
		return 0, someErr
	})

	handled := base.Handle(func(res int, err error) (int, error) {
		require.ErrorIs(t, err, someErr, "Handle should receive original error")
		require.Equal(t, 0, res, "Handle should receive zero result on error")
		return 999, nil
	})

	handled.Run()
	v, err := handled.Wait()

	require.NoError(t, err, "Handle() should be able to recover from error")
	require.Equal(t, 999, v, "Handle() should return recovered value")
}
