# pipers 

Parallelism helper powered by generics.

[![pipers status](https://github.com/kozhurkin/pipers/actions/workflows/tests.yml/badge.svg)](https://github.com/kozhurkin/pipers/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/kozhurkin/pipers)](https://goreportcard.com/report/github.com/kozhurkin/pipers)
[![codecov](https://codecov.io/gh/kozhurkin/pipers/graph/badge.svg?token=TTDJSWUO7W)](https://codecov.io/gh/kozhurkin/pipers)
[![GitHub Release](https://img.shields.io/github/release/kozhurkin/pipers.svg)]()

#### Why is pipers better than sync.WaitGroup or errgroup.Group?
✔ Because pipers can catch errors.\
✔ Pipers knows how to return a caught error immediately, without waiting for a response from parallel goroutines.\
✔ Pipers allows you to set the number of errors you want to return. `.FirstNErrors(n)` `.ErrorsAll()`\
✔ Pipers knows how to take a context as an argument and handle its termination. `.Context(ctx)`\
✔ Pipers knows how to limit the number of simultaneously executed goroutines. `.Concurrecy(n)`\
✔ Pipers allow you to write cleaner and more compact code.

Installing
----------

	go get github.com/kozhurkin/pipers

Example
-------

``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    args := []string{"pipers", "is", "a", "parallelism", "helper", "powered", "by", "generics"}

    pp := pipers.FromArgs(args, func(i int, word string) (int, error) {
        length := len(word)
        sleeptime := time.Duration(length) * time.Second
        <-time.After(sleeptime)
        return length, nil
    })

    results, err := pp.Resolve()

    fmt.Println(results, err, time.Since(ts))
    // [6 2 1 11 6 7 2 8] <nil> 11.00s
}
```

Usage
-----

✔ [`pipers.FromFuncs(...funcs)`](#pipersfromfuncsfuncs)
✔ [`pipers.FromFuncsCtx(...funcs)`](#pipersfromfuncsctxfuncs)\
✔ [`pipers.FromArgs(args, handler)`](#pipersfromargsargs-handler)
✔ [`pipers.FromArgsCtx(args, handler)`](#pipersfromargsctxargs-handler)\
✔ [`pipers.Ref(&v, func)`](#pipersrefv-func)\
✔ [`pp.Concurrency(n)`](#ppconcurrencyn)\
✔ [`pp.Context(ctx)`](#ppcontextctx)\
✔ [`pp.FirstNErrors(n)`](#ppfirstnerrorsn)\
✔ [`pp.ErrorsAll()`](#pperrorsall)\
✔ [`pp.Ctx()`](#ppctx)

### pipers.FromFuncs(...funcs)
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()

    //...........vvvvvvvvv
    pp := pipers.FromFuncs(
        func() (interface{}, error) { time.Sleep(2 * time.Second); return "Happy", nil },
        func() (interface{}, error) { time.Sleep(0 * time.Second); return []byte("New"), nil },
        func() (interface{}, error) { time.Sleep(2 * time.Second); return bytes.NewBufferString("Year"), nil },
        func() (interface{}, error) { time.Sleep(4 * time.Second); return byte('!'), nil },
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
}
```

### pipers.FromFuncsCtx(...funcs)
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()

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
}
```

### pipers.FromArgs(args, handler)
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    args := []int{1, 2, 3, 4, 5}

    //...........vvvvvvvv
    pp := pipers.FromArgs(args, func(i int, a int) (int, error) {
        <-time.After(time.Duration(i) * time.Second)
        return a * a, nil
    })

    results, err := pp.Resolve()

    fmt.Println(results, err, time.Since(ts))
    // [1 4 9 16 25] <nil> 4.00s
}
```

### pipers.FromArgsCtx(args, handler)
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

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
        return uint8(fact), nil
    })

    results, err := pp.Concurrency(3).Resolve()

    fmt.Println(results, err, time.Since(ts))
    // [1 2 6 24 120 208 0 0 0] uint8 overflow 8.00s
    // break 7! iterations skipped: 1
    // break 8! iterations skipped: 4
}
```

### pipers.Ref(&v, func)
Helper for specifying values by pointer.
It can be more convenient than type conversion.
``` golang
import github.com/kozhurkin/pipers

func main() {
    var a *http.Response
    var b []byte
    var c int

    pp := pipers.FromFuncs(
    //.........vvv
        pipers.Ref(&a, func() (*http.Response, error) { return http.Get("https://github.com") }),
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
}
```

### pp.Concurrency(n)
Allows you to limit `n` the number of simultaneously executed goroutines.\
`1` - means that goroutines will be executed one by one.\
`0` - means that all the goroutines will run at once simultaneously in parallel.
``` golang
import github.com/kozhurkin/pipers

func main() {
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
        res, err := http.Get(url)
        if err != nil {
            return -1, err
        }
        return res.StatusCode, err
    })

    // vvvvvvvvvvv
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
```

### pp.Context(ctx)
Allows you to take a context as an argument and handle its termination.\
Сan be used, for example, to specify a timeout `context.WithTimeout`.
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    delays := []int{3, 6, 2, 4, 1, 5}

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pp := pipers.FromArgs(delays, func(i int, delay int) (float64, error) {
        fmt.Printf("func(%v, %v) %v\n", i, delay, time.Since(ts))
        time.Sleep(time.Duration(delay) * time.Second)
        return float64(delay), nil
    })
    // vvvvvvv
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
```

### pp.FirstNErrors(n)
Allows you to set `n` the number of errors you want to return.\
`0` - will return any errors that have occurred.\
If there were no errors, the method returns `nil`.
``` golang
import github.com/kozhurkin/pipers

func main() {
    data := []string{"one", "two", "three", "four", "five", "six", "seven"}

    pp := pipers.FromArgs(data, func(i int, value string) (int, error) {
        if i%2 == 0 {
            return -1, errors.New(value)
        }
        return 1, nil
    }).Concurrency(1)

    //.........vvvvvvvvvvvv
    errs := pp.FirstNErrors(2)
    results := pp.Results()

    fmt.Println(results, errs)
    // [-1 1 -1 0 0 0 0] [one three]
}
```

### pp.ErrorsAll()
Returns all errors that occurred. Similar to `pp.FirstNErrors(0)`.\
Note that if the context is canceled, only the errors that were received at the moment of cancelation will be returned.
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    data := []string{"one", "two", "three", "four", "five", "six", "seven"}

    pp := pipers.FromArgs(data, func(i int, value string) (int, error) {
        <-time.After(time.Duration(i+1) * time.Second)
        if i%2 == 0 {
            return -1, errors.New(value)
        }
        return 1, nil
    })

    ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
    defer cancel()

    //......................vvvvvvvvv
    errs := pp.Context(ctx).ErrorsAll()
    results := pp.Results()

    fmt.Println(results, errs, time.Since(ts))
    // [-1 1 -1 1 -1 1 0] [one three five context deadline exceeded] 6.00s
}
```


### pp.Ctx()
Returns a wrapped context that will be immediately canceled if an error occurs.\
Can be used to interrupt execution in parallel goroutines.
``` golang
import github.com/kozhurkin/pipers

func main() {
    ts := time.Now()
    data := []int{2, 4, 8, 10, 12}

    var pp *pipers.PiperSolver[uint8]
    pp = pipers.FromArgs(data, func(_ int, n int) (uint8, error) {
        fact := 1
        for i := 2; i <= n; i++ {
            select {
            case <-time.After(time.Second):
                if fact *= i; fact > math.MaxUint8 {
                    return uint8(fact), errors.New("uint8 overflow")
                }
            //........vvv
            case <-pp.Ctx().Done():
                fmt.Printf("break %v! iterations skipped: %v\n", n, n-i)
                return uint8(fact), nil
            }
        }
        return uint8(fact), nil
    })

    results, err := pp.Concurrency(3).Resolve()

    fmt.Println(results, err, time.Since(ts))
    // [2 24 208 0 0] uint8 overflow 5.00s
    // break 10! iterations skipped: 4
    // break 12! iterations skipped: 8
}
```