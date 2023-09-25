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

func ParallelWithResult[TIN any, TOUT any](
	inputQueue <-chan TIN,
	outputQueue chan<- TOUT,
	threadCount int,
	workerFunc func(workItem TIN) TOUT,
) {

	if threadCount <= 0 {
		threadCount = runtime.NumCPU()
	}

	var waitGroup sync.WaitGroup

	for cpu := 0; cpu < threadCount; cpu++ {
		waitGroup.Add(1)
		go func(channel <-chan TIN) {
			for {
				var workItem, more = <-channel

				if more {
					var result = workerFunc(workItem)
					outputQueue <- result
				} else {
					waitGroup.Done()
					return
				}
			}
		}(inputQueue)
	}

	waitGroup.Wait()
}
