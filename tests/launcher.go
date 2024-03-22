package tests

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

type ProcessInfo [5]*struct {
	Delay int
	Err   error
}

func (pi ProcessInfo) Duration(c int) int {
	m := make(map[int]int)
	if c == 0 || c > len(pi) {
		c = len(pi)
	}
	for i, p := range pi {
		m[i%c] += p.Delay
	}
	maxv := 0
	for _, v := range m {
		if v > maxv {
			maxv = v
		}
	}
	return maxv
}

type Expectations []*struct {
	Concurrency int
	Iterations  int32
	Duration    int
	Result
	Error error
}

type Tasks []*struct {
	Desc string
	Args [5]int
	ProcessInfo
	CancelAfter int
	TimeUnit    time.Duration
	Expectations
}

type Launcher struct {
	T *testing.T
	Tasks
	Handler func(context.Context, []int, func(int, int) (int, error), int) ([]int, error)
}

const DURATION_FAILT = 5

func (l Launcher) Pick(ti int, ei int) *Launcher {
	if ti != -1 {
		l.Tasks = l.Tasks[ti : ti+1]
	}
	for _, t := range l.Tasks {
		if ei != -1 {
			t.Expectations = t.Expectations[ei : ei+1]
		}
	}
	return &l
}

func (l Launcher) Run() *Launcher {
	for _, task := range l.Tasks {
		task := task
		//for _, c := range []int{task.Concurrency} {
		for _, expect := range task.Expectations {
			ctx := context.Background()
			if task.CancelAfter != 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(task.CancelAfter)*task.TimeUnit)
				defer cancel()
			}
			wait := time.After(time.Duration(task.ProcessInfo.Duration(expect.Concurrency)) * task.TimeUnit)
			var cnt int32
			ts := time.Now()
			result, err := l.Handler(ctx, task.Args[:5], func(i int, arg int) (int, error) {
				defer func() {
					atomic.AddInt32(&cnt, 1)
				}()
				pi := task.ProcessInfo[i]
				<-time.After(time.Duration(pi.Delay) * task.TimeUnit)
				if pi.Err != nil {
					return 0, pi.Err
				}
				return arg * arg, nil
			}, expect.Concurrency)

			duration := int(time.Since(ts) / task.TimeUnit)
			diff := expect.Duration - duration

			<-wait
			<-time.After(time.Duration(30*DURATION_FAILT) * task.TimeUnit)

			l.T.Log(fmt.Sprintf(
				"%v :  c=%v, %v \t %v %v \t %v %v      %v %v      %v (%v)",
				task.Desc,
				expect.Concurrency,
				task.CancelAfter,
				formatBool(atomic.LoadInt32(&cnt) == expect.Iterations),
				atomic.LoadInt32(&cnt),
				formatBool(math.Abs(float64(diff)) < DURATION_FAILT),
				duration,
				formatBool(expect.Result.IsEqual(result)),
				result,
				formatBool(errors.Is(err, expect.Error)),
				err,
			))
		}
		l.T.Log()
	}
	return &l
}

type Result [5]int

func (r Result) IsEqual(m []int) bool {
	//fmt.Println(r.String(), Result(m).String())
	if len(r) != len(m) {
		return false
	}
	for i, _ := range r {
		if r[i] != m[i] {
			return false
		}
	}
	return true
}

func formatBool(b bool) string {
	if b {
		//return "✔"
		return "✅"
	} else {
		//return "✖"
		return "❌"
	}
}
