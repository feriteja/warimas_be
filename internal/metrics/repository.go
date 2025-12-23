package metrics

import (
	"sync/atomic"
	"time"
)

type Counter struct {
	value uint64
}

func (c *Counter) Inc() {
	atomic.AddUint64(&c.value, 1)
}

func (c *Counter) Add(n uint64) {
	atomic.AddUint64(&c.value, n)
}

func (c *Counter) Load() uint64 {
	return atomic.LoadUint64(&c.value)
}

type Timer struct {
	start time.Time
}

func StartTimer() *Timer {
	return &Timer{start: time.Now()}
}

func (t *Timer) Duration() time.Duration {
	return time.Since(t.start)
}
