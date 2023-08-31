package Lib

import (
	"runtime"
	"sync"
)

func Parallel[T any](channel <-chan T, threadCount int, workerFunc func(workItem T)) {

	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	var waitGroup sync.WaitGroup

	for cpu := 0; cpu < threadCount; cpu++ {
		waitGroup.Add(1)
		go func(channel <-chan T) {
			for {
				var workItem, more = <-channel

				if more {
					workerFunc(workItem)
				} else {
					waitGroup.Done()
					return
				}
			}
		}(channel)
	}

	waitGroup.Wait()
}
