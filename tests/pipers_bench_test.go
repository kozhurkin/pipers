package tests

import (
	"context"
	"errors"
	"github.com/kozhurkin/pipers"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var datas = func() [][]int {
	res := [][]int{
		make([]int, 1),
		make([]int, 4),
		make([]int, 16),
		make([]int, 64),
		make([]int, 256),
	}
	for _, data := range res {
		for i := range data {
			data[i] = rand.Intn(10000)
		}
	}
	return res
}()

var seed = time.Now().UnixNano()

func bench(b *testing.B, asyncFunc func(context.Context, []int, func(int, int) (int, error), int) ([]int, error)) {
	ctx := context.Background()
	rand.Seed(seed)
	var errs, iters int32
	for _, data := range datas {
		length := len(data)
		for c := 0; c <= 10; c++ {
			for i := 1; i <= b.N; i++ {
				_, _ = asyncFunc(ctx, data, func(i int, k int) (int, error) {
					rnd := rand.Intn(1000)
					runtime.Gosched()
					//<-time.After(time.Duration(rnd) * time.Nanosecond)
					atomic.AddInt32(&iters, 1)
					if rand.Intn(length) == 0 {
						atomic.AddInt32(&errs, 1)
						return rnd, errors.New("unknown error")
					}
					return rnd, nil
				}, c)
			}
		}
	}
	//fmt.Println("throw", b.N, atomic.LoadInt32(&iters), atomic.LoadInt32(&errs))
}

func BenchmarkAsyncPipers(b *testing.B) {
	bench(b, func(ctx context.Context, args []int, f func(int, int) (int, error), concurrency int) ([]int, error) {
		return pipers.FromArgs(args, f).Context(ctx).Concurrency(concurrency).Resolve()
	})
}
