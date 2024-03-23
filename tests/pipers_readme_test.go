package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/kozhurkin/async"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestReadmeExample(t *testing.T) {
	ts := time.Now()
	args := []string{"pipers", "is", "parallelism", "helper", "powered", "by", "generics"}

	pp := pipers.FromArgs(args, func(i int, word string) (int, error) {
		length := len(word)
		sleep := time.Duration(length) * time.Millisecond
		<-time.After(sleep)
		return length, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [6 2 11 6 7 2 8] <nil> 11.00ms
}

func TestReadmeFromFuncs(t *testing.T) {
	ts := time.Now()
	pp := pipers.FromFuncs(
		func() (string, error) { time.Sleep(2 * time.Millisecond); return "Happy", nil },
		func() (string, error) { time.Sleep(0 * time.Millisecond); return "New", nil },
		func() (string, error) { time.Sleep(2 * time.Millisecond); return "Year", nil },
		func() (string, error) { time.Sleep(4 * time.Millisecond); return "!", nil },
	)

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [Happy New Year !] <nil> 4.00ms
}

func TestReadmeFromFuncs2(t *testing.T) {
	ts := time.Now()
	pp := pipers.FromFuncs(
		func() (interface{}, error) { time.Sleep(2 * time.Second); return "Happy", nil },
		func() (interface{}, error) { time.Sleep(0 * time.Second); return []byte("New"), nil },
		func() (interface{}, error) { time.Sleep(2 * time.Second); return bytes.NewBufferString("Year"), nil },
		func() (interface{}, error) { time.Sleep(4 * time.Second); return byte('!'), nil },
	)

	res, err := pp.Resolve()

	r0, r1, r2, r3 := res[0].(string), res[1].([]byte), res[2].(*bytes.Buffer), res[3].(byte)

	fmt.Println(res, err, time.Since(ts))
	fmt.Println(r0, string(r1), r2.String(), string(r3))
	// [Happy [78 101 119] Year 33] <nil> 4.00s
	// Happy New Year !
}

func TestReadmeFromArgs(t *testing.T) {
	ts := time.Now()
	args := []int{1, 2, 3, 4, 5}

	pp := pipers.FromArgs(args, func(i int, a int) (int, error) {
		time.Sleep(time.Duration(i) * time.Millisecond)
		return a * a, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [1 4 9 16 25] <nil> 4.00ms
}

func TestReadmeRef(t *testing.T) {
	var a *http.Response
	var b []byte
	var c int

	pp := pipers.FromFuncs(
		pipers.Ref(&a, func() (*http.Response, error) { return http.Get("https://github.com") }),
		pipers.Ref(&b, func() ([]byte, error) { return os.ReadFile("/etc/hosts") }),
		pipers.Ref(&c, func() (int, error) { return 777, nil }),
	)

	results, _ := pp.Resolve()

	fmt.Printf("results: %T, %v \n", results, len(results))
	fmt.Printf("      a: %T, %v \n", a, a.Status)
	fmt.Printf("      b: %T, %v \n", b, len(b))
	fmt.Printf("      c: %T, %v \n", c, c)

	// results: []interface {}, 3
	//       a: *http.Response, 200 OK
	//       b: []uint8, 213
	//       c: int, 777

	// without .Ref() you would have to do type conversion for slice elements
	// a := results[0].(*http.Response)
	// b := results[1].([]byte)
	// c := results[2].(int)
}

func TestReadmeContext(t *testing.T) {
	ts := time.Now()
	delays := []int{3, 6, 2, 4, 1, 5, 1}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	pp := pipers.FromArgs(delays, func(i int, delay int) (float64, error) {
		fmt.Printf("func(%v, %v) %v\n", i, delay, time.Since(ts))
		time.Sleep(time.Duration(delay) * time.Millisecond)
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

func TestReadmeConcurrency(t *testing.T) {
	ts := time.Now()
	urls := []string{
		"https://nodejs.org",
		"https://go.dev",
		"https://vuejs.org",
		"https://clickhouse.com",
		"https://invalid.link",
		"https://github.com",
	}

	pp := pipers.FromArgs(urls, func(i int, url string) (int, error) {
		fmt.Printf("func(%v, %v) %v\n", i, url, time.Since(ts))
		res, err := http.Get(url)
		if err != nil {
			return -1, err
		}
		return res.StatusCode, err
	})

	pp.Concurrency(2)

	results, err := pp.Resolve()

	fmt.Println(time.Since(ts), results, err)
	// func(1, https://go.dev) 243.416µs
	// func(0, https://nodejs.org) 567.041µs
	// func(2, https://vuejs.org) 362.966375ms
	// func(3, https://clickhouse.com) 371.109291ms
	// func(4, https://invalid.link) 660.3065ms
	// 661.806125ms [200 200 0 200 -1 0] Get "https://invalid.link": dial tcp: lookup invalid.link: no such host
}

func TestReadmeFirstNErrors(t *testing.T) {
	data := []string{"one", "two", "three", "four", "five", "six", "seven"}

	pp := pipers.FromArgs(data, func(i int, value string) (int, error) {
		if i%2 == 0 {
			return -1, errors.New(value)
		}
		return 1, nil
	}).Concurrency(1)

	errs := pp.FirstNErrors(2)
	results := pp.Results()

	fmt.Println(results, errs)
	// [-1 1 -1 0 0 0 0] [one three]
}

func TestReadmeErrorsAll(t *testing.T) {
	ts := time.Now()
	data := []string{"one", "two", "three", "four", "five", "six", "seven"}

	pp := pipers.FromArgs(data, func(i int, value string) (int, error) {
		<-time.After(time.Duration(i+1) * time.Millisecond)
		if i%2 == 0 {
			return -1, errors.New(value)
		}
		return 1, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Millisecond)
	defer cancel()

	errs := pp.Context(ctx).ErrorsAll()
	results := pp.Results()

	fmt.Println(results, errs, time.Since(ts))
	// [-1 1 -1 1 -1 1 0] [one three five context deadline exceeded] 6.001158667s
}
