package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kozhurkin/pipers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var httpclient = http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestReadmeExample(t *testing.T) {
	ts := time.Now()
	args := []string{"pipers", "is", "a", "parallelism", "helper", "powered", "by", "generics"}

	pp := pipers.FromArgs(args, func(i int, word string) (int, error) {
		length := len(word)
		sleeptime := time.Duration(length) * time.Millisecond
		<-time.After(sleeptime)
		return length, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [6 2 11 6 7 2 8] <nil> 11.00ms

	assert.Nil(t, err)
	assert.InDelta(t, 11, int(time.Since(ts).Milliseconds()), 1)
}

func TestReadmeFromFuncs(t *testing.T) {
	ts := time.Now()

	pp := pipers.FromFuncs(
		func() (interface{}, error) { time.Sleep(2 * time.Millisecond); return "Happy", nil },
		func() (interface{}, error) { time.Sleep(0 * time.Millisecond); return []byte("New"), nil },
		func() (interface{}, error) {
			time.Sleep(2 * time.Millisecond)
			return bytes.NewBufferString("Year"), nil
		},
		func() (interface{}, error) { time.Sleep(4 * time.Millisecond); return byte('!'), nil },
	)

	res, err := pp.Resolve()

	r0 := res[0].(string)
	r1 := res[1].([]byte)
	r2 := res[2].(*bytes.Buffer)
	r3 := res[3].(byte)

	fmt.Println(res, err, time.Since(ts))
	fmt.Println(r0, string(r1), r2.String(), string(r3))
	// [Happy [78 101 119] Year 33] <nil> 4.00s
	// Happy New Year !

	assert.Nil(t, err)
	assert.InDelta(t, 4, int(time.Since(ts).Milliseconds()), 1)
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

	assert.Nil(t, err)
	assert.InDelta(t, 4, int(time.Since(ts).Milliseconds()), 1)
}

func TestReadmeRef(t *testing.T) {
	var a *http.Response
	var b []byte
	var c int

	pp := pipers.FromFuncs(
		pipers.Ref(&a, func() (*http.Response, error) { return httpclient.Get("https://github.com") }),
		pipers.Ref(&b, func() ([]byte, error) { return exec.Command("uname", "-m").Output() }),
		pipers.Ref(&c, func() (int, error) { return 777, nil }),
	)

	results, err := pp.Resolve()

	fmt.Println("results:", reflect.TypeOf(results), results, err)
	fmt.Println("a:", reflect.TypeOf(a), a.StatusCode)
	fmt.Println("b:", reflect.TypeOf(b), string(b))
	fmt.Println("c:", reflect.TypeOf(c), c)

	// results: []interface {} [0xc000178000 [97 114 109 54 52 10] 777] <nil>

	// a: *http.Response 200
	// b: []uint8 arm64
	// c: int 777

	// without .Ref() you would have to do type conversion for slice elements
	// a := results[0].(*http.Response)
	// b := results[1].([]byte)
	// c := results[2].(int)

	assert.Nil(t, err)
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

	// func(0, 3) 0.00s
	// func(1, 6) 0.00s
	// func(2, 2) 0.00s
	// func(3, 4) 2.00s
	// func(4, 1) 3.00s
	// func(5, 5) 4.00s
	// [3 0 2 0 1 0] context deadline exceeded 5.00s

	assert.Equal(t, context.DeadlineExceeded, err)
	assert.InDelta(t, 5, int(time.Since(ts).Milliseconds()), 1)
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
		defer func() {
			fmt.Println(i, url, time.Since(ts))
		}()
		resp, err := httpclient.Get(url)
		if err != nil {
			return -1, err
		}
		return resp.StatusCode, err
	})

	pp.Concurrency(2)

	results, err := pp.Resolve()

	fmt.Println(time.Since(ts), results, err)
	// 1 https://go.dev 270.200083ms
	// 2 https://vuejs.org 360.896208ms
	// 0 https://nodejs.org 440.650167ms
	// 4 https://invalid.link 442.175792ms
	// 442.23575ms [200 200 200 0 -1 0] Get "https://invalid.link": dial tcp: lookup invalid.link: no such host

	assert.Equal(t, -1, results[4])
	assert.NotNil(t, err)

	fmt.Println("<-pp.Tail()")
	<-pp.Tail()
	fmt.Println(time.Since(ts))
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

	assert.Equal(t, 2, len(errs))
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
	// [-1 1 -1 1 -1 1 0] [one three five context deadline exceeded] 6.00s

	assert.Equal(t, 4, len(errs))
	assert.Equal(t, context.DeadlineExceeded, errs[3])
	assert.InDelta(t, 6, int(time.Since(ts).Milliseconds()), 1)

	<-pp.Tail()
	fmt.Println(time.Since(ts))
}

func TestReadmeNoErrors(t *testing.T) {
	data := make([]int, 9)

	pp := pipers.FromArgs(data, func(i int, v int) (int, error) { return 1, nil }).Concurrency(2)

	errs := pp.FirstNErrors(2)
	results := pp.Results()

	fmt.Println(results, errs, errs == nil)
	// [1 1 1 1 1 1 1] []

	assert.Equal(t, 1, results[8])
	assert.Nil(t, errs)
}

func TestReadmeNotEnoughtErrors(t *testing.T) {
	data := make([]error, 9)
	data[0] = errors.New("throw")

	pp := pipers.FromArgs(data, func(i int, e error) (error, error) { return e, e })

	errs := pp.Concurrency(2).FirstNErrors(2)
	results := pp.Results()

	fmt.Println(results, len(results), errs)
	fmt.Println(results.Shift(), len(results))
	// [throw <nil> <nil> <nil> <nil> <nil> <nil> <nil> <nil>] 9 [throw]
	// throw 8

	assert.Equal(t, len(data)-1, len(results))
}

func TestReadmeFromArgsCtx(t *testing.T) {
	var cnt int32
	ts := time.Now()
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	//...........vvvvvvvvvvv............vvv
	pp := pipers.FromArgsCtx(data, func(ctx context.Context, _ int, n int) (uint8, error) {
		fact := 1
		for i := 2; i <= n; i++ {
			select {
			case <-time.After(time.Millisecond):
				if fact *= i; fact > math.MaxUint8 {
					return uint8(fact), errors.New("uint8 overflow")
				}
			case <-ctx.Done():
				fmt.Printf("break %v! iterations skipped: %v\n", n, n-i)
				return uint8(fact), nil
			}
		}
		atomic.AddInt32(&cnt, 1)
		return uint8(fact), nil
	})

	results, err := pp.Concurrency(3).Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [1 2 6 24 120 208 0 0 0] uint8 overflow 8.00s
	// break 7! iterations skipped: 1
	// break 8! iterations skipped: 4

	<-time.After(10 * time.Millisecond)
	assert.Equal(t, 5, int(atomic.LoadInt32(&cnt)))
}

func TestReadmeFromFuncsCtx(t *testing.T) {
	ts := time.Now()

	//...........vvvvvvvvvvvv......vvv
	pp := pipers.FromFuncsCtx(
		func(ctx context.Context) (bool, error) {
			<-time.After(3 * time.Millisecond)
			return true, errors.New("throw")
		},
		func(ctx context.Context) (bool, error) {
			ticker := time.NewTicker(time.Millisecond)
			for {
				select {
				case <-ticker.C:
					fmt.Println("tick")
				case <-ctx.Done():
					fmt.Println("break")
					ticker.Stop()
					return true, nil
				}
			}
		},
	)

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// tick
	// tick
	// break
	// [true false] throw 3.00s

	assert.InDelta(t, 3, int(time.Since(ts).Milliseconds()), 1)
}

func TestReadmeTail(t *testing.T) {
	ts := time.Now()
	args := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	pp := pipers.FromArgs(args, func(i int, v int) (int, error) {
		<-time.After(5 * time.Millisecond)
		return v, nil
	})

	err := pp.Context(ctx).Concurrency(5).FirstError()
	fmt.Println(pp.Results(), err, time.Since(ts))

	//...vvvv
	<-pp.Tail()
	results := pp.Results()
	fmt.Println(results, time.Since(ts))

	// [0 0 0 0 0 0 0 0 0] context deadline exceeded 1.00s
	// [1 2 3 4 5 0 0 0 0] 5.00s

	assert.InDelta(t, 5, int(time.Since(ts).Milliseconds()), 1)
	assert.Equal(t, 5, results[4])

	fmt.Println(pipers.Map(args, results))
}

func TestReadmeZeroLength(t *testing.T) {
	ts := time.Now()
	args := []int{}

	pp := pipers.FromArgs(args, func(i int, v int) (int, error) {
		<-time.After(5 * time.Millisecond)
		return v, nil
	})

	err := pp.Concurrency(5).FirstError()
	fmt.Println(pp.Results(), err, time.Since(ts))

	<-pp.Tail()
	results := pp.Results()
	fmt.Println(results, time.Since(ts))

	fmt.Println(pipers.Map(args, results))
	fmt.Println(pipers.Map(args, []int{5}))
	fmt.Println(pipers.Flatten([][]int{{777}, {999}}))
}

func TestReadmeCheck(t *testing.T) {
	carriers := []int{1, 2, 3, 4, 5}
	errs := make([]error, len(carriers))
	quotes := make([]int, len(carriers))
	var wg sync.WaitGroup
	for i, c := range carriers {
		wg.Add(1)
		go func(i int, c int) {
			defer wg.Done()
			quotes[i], errs[i] = c*c, nil
		}(i, c)
	}
	wg.Wait()
	fmt.Println(errs, quotes)
}
