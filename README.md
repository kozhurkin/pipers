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

Usage
-----

``` golang
import github.com/kozhurkin/async/pipers

func main() {
    ts := time.Now()
    delays := []int{3, 6, 2, 4, 1, 5}

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
```

#### pipers.FromFuncs()
``` golang
import github.com/kozhurkin/async/pipers

func main() {
	ts := time.Now()

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

#### pipers.FromSlice()
``` golang
import github.com/kozhurkin/async/pipers

func main() {
	ts := time.Now()
	args := []int{1, 2, 3, 4, 5}

	pp := pipers.FromSlice(args, func(i int, a int) (int, error) {
		<-time.After(time.Duration(i) * time.Second)
		return a * a, nil
	})

	results, err := pp.Resolve()

	fmt.Println(results, err, time.Since(ts))
	// [1 4 9 16 25] <nil> 4.00s
}
```