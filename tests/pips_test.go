package tests

import (
	"fmt"
	"github.com/kozhurkin/pipers/pips"
	"testing"
	"time"
)

func TestPips(t *testing.T) {
	ts := time.Now()
	pa := pips.NewPip(func() int { <-time.After(1 * time.Second); return 1 })
	pb := pips.NewPip(func() int { <-time.After(2 * time.Second); return 2 })
	a, b := <-pa, <-pb
	fmt.Println(a, b, time.Since(ts))
	delta := int(time.Now().Sub(ts).Seconds())
	if delta != 2 {
		t.Fatal("Should complete in 2 seconds")
	}
	if a != 1 || b != 2 {
		t.Fatal("Wrong return values")
	}
	return
}
func TestPipsFromFuncs(t *testing.T) {
	ts := time.Now()
	res := pips.FromFuncs(
		func() int { <-time.After(1 * time.Second); return 1 },
		func() int { <-time.After(2 * time.Second); return 2 },
	)
	fmt.Println(res, time.Since(ts))
	delta := int(time.Now().Sub(ts).Seconds())
	if delta != 2 {
		t.Fatal("Should complete in 2 seconds")
	}
	if res[0] != 1 || res[1] != 2 {
		t.Fatal("Wrong return values")
	}
	return
}

func TestPipsFromArgs(t *testing.T) {
	ts := time.Now()
	res := pips.FromArgs([]int{1, 2}, func(i int, a int) int {
		<-time.After(time.Duration(a) * time.Second)
		return a
	})
	fmt.Println(res, time.Since(ts))
	delta := int(time.Now().Sub(ts).Seconds())
	if delta != 2 {
		t.Fatal("Should complete in 2 seconds")
	}
	if res[0] != 1 || res[1] != 2 {
		t.Fatal("Wrong return values")
	}
	return
}
