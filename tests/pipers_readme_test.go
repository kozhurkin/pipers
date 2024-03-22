package tests

import (
	"fmt"
	"github.com/kozhurkin/async"
	"testing"
	"time"
)

func TestReadmeFromFuncs(t *testing.T) {
	ts := time.Now()

	pp := pipers.FromFuncs(
		func() (string, error) { <-time.After(2 * time.Second); return "Happy", nil },
		func() (string, error) { <-time.After(3 * time.Second); return "New", nil },
		func() (string, error) { <-time.After(1 * time.Second); return "Year!", nil },
	)

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts)) // [Happy New Year!] <nil> 3.00s
}

func TestReadmeFromSlice(t *testing.T) {
	ts := time.Now()
	args := []int{1, 2, 3, 4, 5}

	pp := pipers.FromSlice(args, func(i int, a int) (int, error) {
		<-time.After(time.Duration(i) * time.Second)
		return a * a, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts)) // [1 4 9 16 25] <nil> 4.00s
}
