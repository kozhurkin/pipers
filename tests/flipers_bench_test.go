package tests

import (
	"context"
	"testing"

	"github.com/kozhurkin/pipers"
)

func BenchmarkAsyncFlipers(b *testing.B) {
	bench(b, func(ctx context.Context, args []int, f func(int, int) (int, error), concurrency int) ([]int, error) {
		var fs pipers.FliperSolver[int]

		fs.Context(ctx).Concurrency(concurrency)

		for i, arg := range args {
			i, arg := i, arg
			fs.AddFunc(func() (int, error) {
				return f(i, arg)
			})
		}

		return fs.Resolve()
	})
}
