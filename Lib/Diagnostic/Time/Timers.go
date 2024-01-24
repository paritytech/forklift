package Time

import (
	"sync"
	"time"
)

var timers = make(map[string]time.Time)
var lock sync.RWMutex

type ForkliftTimer struct {
}

func Start(name string) {
	lock.Lock()
	defer lock.Unlock()

	timers[name] = time.Now()
}

func Stop(name string) time.Duration {
	lock.Lock()
	defer lock.Unlock()

	var start = timers[name]
	delete(timers, name)
	return time.Since(start)
}

func (ft *ForkliftTimer) MeasureTime(worker func()) time.Duration {
	var start = time.Now()
	worker()
	return time.Since(start)
}
