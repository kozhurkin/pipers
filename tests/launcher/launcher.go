package launcher

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

type Flow [5]*struct {
	Delay int
	Error error
}

func (pi Flow) MaxDuration(c int) int {
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
	Flow
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
		for _, expect := range task.Expectations {
			ctx := context.Background()
			if task.CancelAfter != 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(task.CancelAfter)*task.TimeUnit)
				defer cancel()
			}
			var cnt int32
			waitchan := time.After(time.Duration(task.Flow.MaxDuration(expect.Concurrency)) * task.TimeUnit)
			ts := time.Now()

			result, err := l.Handler(ctx, task.Args[:5], func(i int, arg int) (int, error) {
				defer func() {
					atomic.AddInt32(&cnt, 1)
				}()
				pi := task.Flow[i]
				<-time.After(time.Duration(pi.Delay) * task.TimeUnit)
				if pi.Error != nil {
					return 0, pi.Error
				}
				return arg * arg, nil
			}, expect.Concurrency)

			duration := int(time.Since(ts) / task.TimeUnit)
			diff := expect.Duration - duration

			<-waitchan
			<-time.After(time.Duration(3*DURATION_FAILT) * task.TimeUnit)

			cntOk := atomic.LoadInt32(&cnt) == expect.Iterations
			durationOk := math.Abs(float64(diff)) < DURATION_FAILT
			resultOk := expect.Result.IsEqual(result)
			errorOk := errors.Is(err, expect.Error)

			l.T.Log(fmt.Sprintf(
				"%v :  c=%v, %v \t %v %v \t %v %v      %v %v      %v (%v)",
				task.Desc,
				expect.Concurrency,
				task.CancelAfter,
				formatBool(cntOk),
				atomic.LoadInt32(&cnt),
				formatBool(durationOk),
				duration,
				formatBool(resultOk),
				result,
				formatBool(errorOk),
				err,
			))

			ok := cntOk && durationOk && resultOk && errorOk

			if !ok {
				l.T.Error("something is wrong!")
			}

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
	for i := range r {
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
