package Time

import (
	"sync"
	"time"
)

type ForkliftTimer struct {
	timers map[string]time.Time
	lock   sync.RWMutex
}

func NewForkliftTimer() *ForkliftTimer {
	return &ForkliftTimer{
		timers: make(map[string]time.Time),
		lock:   sync.RWMutex{},
	}
}

func (timer *ForkliftTimer) Start(name string) {
	timer.lock.Lock()
	defer timer.lock.Unlock()

	timer.timers[name] = time.Now()
}

func (timer *ForkliftTimer) Stop(name string) time.Duration {
	timer.lock.Lock()
	defer timer.lock.Unlock()

	var start = timer.timers[name]
	delete(timer.timers, name)
	return time.Since(start)
}

func (timer *ForkliftTimer) MeasureTime(worker func()) time.Duration {
	var start = time.Now()
	worker()
	return time.Since(start)
}
