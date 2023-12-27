package limitedconcurrent

import "sync"

type empty = struct{}

func LaunchLimited[T any](retrievers []func(chan<- T), limit int) []T {
	outputChan := make(chan T, len(retrievers))
	go manageLaunch(outputChan, retrievers, limit)

	values := make([]T, 0, len(retrievers))
	for value := range outputChan {
		values = append(values, value)
	}
	return values
}

func manageLaunch[T any](outputChan chan<- T, retrievers []func(chan<- T), limit int) {
	guard := make(chan empty, limit)
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(retrievers))
	for _, retriever := range retrievers {
		guard <- empty{}           // take a concurrent place
		retrieverCopy := retriever // avoid closure capture
		go func() {
			retrieverCopy(outputChan)
			<-guard // return the concurrent place
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()  // all work is done
	close(outputChan) // no more sending
}
