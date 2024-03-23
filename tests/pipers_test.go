package tests

import (
	"context"
	"errors"
	"github.com/kozhurkin/pipers"
	"github.com/kozhurkin/pipers/tests/launcher"
	"testing"
	"time"
)

const TIME_UNIT = 3 * time.Millisecond

var throw = errors.New("throw error")
var throw2 = errors.New("throw error (2)")

var tasks = launcher.Tasks{
	{
		Desc: "SUCCESS launch          ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, nil},
			{30, nil},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 0,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 5, 245, launcher.Result{1, 4, 9, 16, 25}, nil},
			{2, 5, 125, launcher.Result{1, 4, 9, 16, 25}, nil},
			{6, 5, 100, launcher.Result{1, 4, 9, 16, 25}, nil},
		},
	},
	{
		Desc: "SUCCESS DEADLINE        ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, nil},
			{30, nil},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 90,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 2, 90, launcher.Result{1, 0, 0, 0, 0}, context.DeadlineExceeded},
			{2, 4, 90, launcher.Result{1, 0, 9, 0, 0}, context.DeadlineExceeded},
			{6, 5, 90, launcher.Result{1, 0, 9, 16, 25}, context.DeadlineExceeded},
		},
	},
	{
		Desc: "DEADLINE before THROW   ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, throw},
			{30, nil},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 90,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 2, 90, launcher.Result{1, 0, 0, 0, 0}, context.DeadlineExceeded},
			{2, 4, 90, launcher.Result{1, 0, 9, 0, 0}, context.DeadlineExceeded},
			{6, 5, 90, launcher.Result{1, 0, 9, 16, 25}, context.DeadlineExceeded},
		},
	},
	{
		Desc: "THROW 1 error simple    ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, throw},
			{30, nil},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 0,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 2, 150, launcher.Result{1, 0, 0, 0, 0}, throw},
			{2, 4, 100, launcher.Result{1, 0, 9, 0, 0}, throw},
			{6, 5, 100, launcher.Result{1, 0, 9, 16, 25}, throw},
		},
	},
	{
		Desc: "THROW 1 before DEADLINE ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, throw},
			{30, nil},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 110,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 2, 110, launcher.Result{1, 0, 0, 0, 0}, context.DeadlineExceeded},
			{2, 4, 100, launcher.Result{1, 0, 9, 0, 0}, throw},
			{6, 5, 100, launcher.Result{1, 0, 9, 16, 25}, throw},
		},
	},
	{
		Desc: "THROW 2 errors following",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{100, throw},
			{30, throw2},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 0,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{1, 2, 150, launcher.Result{1, 0, 0, 0, 0}, throw},
			{2, 3, 80, launcher.Result{1, 0, 0, 0, 0}, throw2},
			{6, 5, 30, launcher.Result{0, 0, 0, 0, 25}, throw2},
		},
	},
	{
		Desc: "LOOONG launch           ",
		Args: [5]int{1, 2, 3, 4, 5},
		Flow: launcher.Flow{
			{50, nil},
			{1000, nil},
			{30, throw},
			{40, nil},
			{25, nil},
		},
		CancelAfter: 0,
		TimeUnit:    TIME_UNIT,
		Expectations: launcher.Expectations{
			{6, 5, 30, launcher.Result{0, 0, 0, 0, 25}, throw},
		},
	},
}

func TestPipers(t *testing.T) {
	launcher.Launcher{t, tasks, func(ctx context.Context, args []int, f func(int, int) (int, error), concurrency int) ([]int, error) {
		return pipers.FromArgs(args, f).Context(ctx).Concurrency(concurrency).Resolve()
	}}.Run()
}
