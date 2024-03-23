# pipers ![pipers status](https://github.com/kozhurkin/pipers/actions/workflows/test.yml/badge.svg)

Parallelism helper powered by generics.

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

✔ [`pipers.FromFuncs(funcs)`](#pipersfromfuncsfuncs)\
✔ [`pipers.FromArgs(args, handler)`](#pipersfromargsargs-handlers)\
✔ [`pipers.Ref(&v, func)`](#pipersrefv-func)\
✔ [`pp.Concurrency(n)`](#ppconcurrencyn)\
✔ [`pp.Context(ctx)`](#ppcontextctx)\
✔ [`pp.FirstNErrors(n)`](#ppfirstnerrorsn)\
✔ [`pp.ErrorsAll()`](#pperrorsall)

### pipers.FromFuncs(funcs)
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

### pipers.FromArgs(args, handlers)
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
        pipers.Ref(&b, func() ([]byte, error) { return os.ReadFile("/etc/hosts") }),
        pipers.Ref(&c, func() (int, error) { return 777, nil }),
    )

    results, _ := pp.Resolve()

    fmt.Printf("results:  %T, %v \n", results, len(results))
    fmt.Printf("a:        %T, %v \n", a, a.Status)
    fmt.Printf("b:        %T, %v \n", b, len(b))
    fmt.Printf("c:        %T, %v \n", c, c)

    // results: []interface {}, 3
    // a:       *http.Response, 200 OK
    // b:       []uint8, 213
    // c:       int, 777

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