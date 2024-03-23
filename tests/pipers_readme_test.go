package tests

import (
	"context"
	"fmt"
	"github.com/kozhurkin/async"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestReadmeExampleUrls(t *testing.T) {
	ts := time.Now()
	urls := []string{
		"https://nodejs.org",
		"https://go.dev",
		"https://vuejs.org",
		"https://clickhouse.com",
		"https://invalid.link",
		"https://github.com",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pp := pipers.FromSlice(urls, func(i int, url string) (int, error) {
		fmt.Printf("func(%v, %v) %v\n", i, url, time.Since(ts))
		res, err := http.Get(url)
		if err != nil {
			return -1, err
		}
		return res.StatusCode, err
	})

	pp.Context(ctx).Concurrency(2)

	results, err := pp.Resolve()

	fmt.Println(time.Since(ts), results, err)
	// func(1, https://go.dev) 243.416µs
	// func(0, https://nodejs.org) 567.041µs
	// func(2, https://vuejs.org) 362.966375ms
	// func(3, https://clickhouse.com) 371.109291ms
	// func(4, https://invalid.link) 660.3065ms
	// 661.806125ms [200 200 0 200 -1 0] Get "https://invalid.link": dial tcp: lookup invalid.link: no such host
}

func TestReadmeRef(t *testing.T) {
	var resp *http.Response
	var file []byte
	var number int

	pp := pipers.FromFuncs(
		pipers.Ref(&resp, func() (*http.Response, error) { return http.Get("https://github.com") }),
		pipers.Ref(&file, func() ([]byte, error) { return os.ReadFile("/etc/hosts") }),
		pipers.Ref(&number, func() (int, error) { return 777, nil }),
	)

	results, _ := pp.Run().Resolve()

	fmt.Printf("results:  %T, %v \n", results, len(results)) // results: []interface {}, 3
	fmt.Printf("resp:     %T, %v \n", resp, resp.Status)     // resp:    *http.Response, 200 OK
	fmt.Printf("file:     %T, %v \n", file, len(file))       // file:    []uint8, 213
	fmt.Printf("number:   %T, %v \n", number, number)        // number:  int, 777
}

func TestReadmeContext(t *testing.T) {
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

	/*	func(0, 3) 0.00s
		func(1, 6) 0.00s
		func(2, 2) 0.00s
		func(3, 4) 2.00s
		func(4, 1) 3.00s
		func(5, 5) 4.00s
		[3 0 2 0 1 0] context deadline exceeded 5.00s
	*/
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
