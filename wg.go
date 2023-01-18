package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type TimeoutWaitGroup struct {
	waitMx *sync.Mutex
	count  *atomic.Int64
}

func NewTimeoutWaitGroup() TimeoutWaitGroup {
	return TimeoutWaitGroup{
		waitMx: &sync.Mutex{},
		count:  &atomic.Int64{},
	}
}

func (t TimeoutWaitGroup) Add(delta int64) {
	t.count.Add(delta)
}

func (t TimeoutWaitGroup) Done() {
	t.Add(-1)
}

func (t TimeoutWaitGroup) Wait(to time.Duration) error {
	t.waitMx.Lock()
	defer t.waitMx.Unlock()

	end := time.Now().Add(to)
	for t.count.Load() > 0 {
		if time.Now().After(end) {
			return errors.New("time exceed")
		}
	}

	return nil
}
