package tests

import (
	"fmt"
	"github.com/kozhurkin/pipers/pips"
	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, delta, 2)
	assert.Equal(t, a, 1)
	assert.Equal(t, b, 2)

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

	assert.Equal(t, delta, 2)
	assert.Equal(t, res[0], 1)
	assert.Equal(t, res[1], 2)

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

	assert.Equal(t, delta, 2)
	assert.Equal(t, res[0], 1)
	assert.Equal(t, res[1], 2)

	return
}
