package tests

import (
	"context"
	"fmt"
	"github.com/kozhurkin/async"
	"testing"
	"time"
)

func TestReadmeExample(t *testing.T) {
	videos := []string{"XqZsoesa55w", "kJQP7kiw5Fk", "RgKAFK5djSk"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pp := pipers.FromSlice(videos, func(i int, vid string) (int, error) {
		views, err := youtube.GetViews(vid)
		if err != nil {
			return 0, err
		}
		return views, nil
	})

	pp.Context(ctx).Concurrency(2)

	results, err := pp.Resolve()

	fmt.Println(results, err) // [14.2e9 8.4e9 6.2e9] <nil>
}

func TestReadmeExample2(t *testing.T) {
	ts := time.Now()
	delays := []int{3, 6, 2, 4, 1, 5, 1}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pp := pipers.FromSlice(delays, func(i int, delay int) (float64, error) {
		fmt.Printf("func(%v, %v) %v\n", i, delay, time.Since(ts))
		time.Sleep(time.Duration(delay) * time.Second)
		return float64(delay), nil
	})

	pp.Context(ctx).Concurrency(3)

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))

	// func(0, 3) 0.00s
	// func(1, 6) 0.00s
	// func(2, 2) 0.00s
	// func(3, 4) 2.00s
	// func(4, 1) 3.00s
	// func(5, 5) 4.00s
	// [3 0 2 0 1 0] context deadline exceeded 5.00s
}

func TestReadmeFromFuncs(t *testing.T) {
	ts := time.Now()
	pp := pipers.FromFuncs(
		func() (string, error) { time.Sleep(2 * time.Second); return "Happy", nil },
		func() (string, error) { time.Sleep(0 * time.Second); return "New", nil },
		func() (string, error) { time.Sleep(2 * time.Second); return "Year", nil },
		func() (string, error) { time.Sleep(4 * time.Second); return "!", nil },
	)

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts)) // [Happy New Year !] <nil> 4.00s
}

func TestReadmeFromSlice(t *testing.T) {
	ts := time.Now()
	args := []int{1, 2, 3, 4, 5}

	pp := pipers.FromSlice(args, func(i int, a int) (int, error) {
		time.Sleep(time.Duration(i) * time.Second)
		return a * a, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts)) // [1 4 9 16 25] <nil> 4.00s
}
