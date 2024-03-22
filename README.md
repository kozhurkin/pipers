# pipers

parallelism helper powered by generics

#### Why is pipers better than sync.WaitGroup or errgroup.Group?
✔ Because pipers can catch errors.\
✔ Pipers knows how to return a caught error immediately, without waiting for a response from parallel goroutines.\
✔ Pipers knows how to take a context as an argument and handle its termination. `.Context()`\
✔ Pipers knows how to limit the number of simultaneously executed goroutines. `.Concurrecy()`\
✔ Pipers allows you to set the number of errors you want to return.\
`.FirstError()` `.FirstNErrors()` `.ErrorsAll()`\
✔ Pipers allow you to write cleaner and more compact code.

Installing
----------

	go get github.com/kozhurkin/pipers

Usage
-----

* `FromFuncs()`
* `FromSlice()`

#### FromFuncs()
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

	fmt.Println(results, err, time.Since(ts)) // [Happy New Year !] <nil> 4.00s
}
```

#### FromSlice()
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

	fmt.Println(results, err, time.Since(ts)) // [1 4 9 16 25] <nil> 4.00s
}
```