package tests

import (
	"context"
	"testing"

	"github.com/kozhurkin/pipers"
	"github.com/kozhurkin/pipers/tests/launcher"
)

// TestFlipers повторяет матрицу сценариев TestPipers,
// но использует FliperSolver (на основе flight.Flight) вместо PiperSolver.
func TestFlipers(t *testing.T) {
	launcher.Launcher{
		T:     t,
		Tasks: tasks,
		Handler: func(ctx context.Context, args []int, f func(int, int) (int, error), concurrency int) ([]int, error) {
			var fs pipers.FliperSolver[int]

			fs.Context(ctx).Concurrency(concurrency)

			for i, arg := range args {
				i, arg := i, arg
				fs.AddFunc(func() (int, error) {
					return f(i, arg)
				})
			}

			return fs.Resolve()
		},
	}.Run()
}
