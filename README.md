# pipers

parallelism helper powered by generics

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
-----

``` golang
import github.com/kozhurkin/async/pipers

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

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pp := pipers.FromSlice(urls, func(i int, url string) (int, error) {
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
```

Usage
-----

* `pipers.FromFuncs(funcs)`
* `pipers.FromArgs(args, handler)`
* `pipers.Ref(&v, func)`

* `pp.Context(ctx)`
* `pp.Concurrency(n)`
* `pp.FirstNErrors()`
* `pp.ErrorsAll()`

#### pipers.FromFuncs(funcs)
``` golang
import github.com/kozhurkin/async/pipers

func main() {
    ts := time.Now()

    //...........vvvvvvvvv
    pp := pipers.FromFuncs(
        func() (string, error) { time.Sleep(2 * time.Second); return "Happy", nil },
        func() (string, error) { time.Sleep(0 * time.Second); return "New", nil },
        func() (string, error) { time.Sleep(2 * time.Second); return "Year", nil },
        func() (string, error) { time.Sleep(4 * time.Second); return "!", nil },
    )

    results, err := pp.Resolve()

    fmt.Println(results, err, time.Since(ts))
    // [Happy New Year !] <nil> 4.00s
}
```

#### pipers.FromArgs(args, handlers)
``` golang
import github.com/kozhurkin/async/pipers

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

#### pipers.Ref(&v, func)
Helper for specifying values by pointer. it can be more convenient than type conversion.
``` golang
import github.com/kozhurkin/async/pipers

func main() {
    var resp *http.Response
    var file []byte
    var number int

    pp := pipers.FromFuncs(
    //.........vvv
        pipers.Ref(&resp, func() (*http.Response, error) { return http.Get("https://github.com") }),
        pipers.Ref(&file, func() ([]byte, error) { return os.ReadFile("/etc/hosts") }),
        pipers.Ref(&number, func() (int, error) { return 777, nil }),
    )

    results, _ := pp.Run().Resolve()

    fmt.Printf("results:  %T, %v \n", results, len(results))
    fmt.Printf("resp:     %T, %v \n", resp, resp.Status)
    fmt.Printf("file:     %T, %v \n", file, len(file))
    fmt.Printf("number:   %T, %v \n", number, number)

    // results: []interface {}, 3
    // resp:    *http.Response, 200 OK
    // file:    []uint8, 213
    // number:  int, 777
    
    // without .Ref() you would have to do type conversion for slice elements
    // resp := res[0].(*http.Response)
    // file := res[1].([]byte)
    // number := res[2].(int)
}
```

#### pp.Context(ctx)
Allows you to take a context as an argument and handle its termination. Сan be used, for example, to specify a timeout `context.WithTimeout`.

``` golang
import github.com/kozhurkin/async/pipers

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
#### pp.Concurrency(n)

Allows you to limit `n` the number of simultaneously executed goroutines.
`1` - means that goroutines will be executed one by one. `0` - means that all the goroutines will run at once simultaneously in parallel.
``` golang

```
