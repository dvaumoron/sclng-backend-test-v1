package limitedconcurrent

import "sync"

type empty = struct{}

func LaunchLimited[T any](senders []func(chan<- T), limit int) []T {
	outputChan := make(chan T, len(senders))
	go manageLaunch(outputChan, senders, limit)

	values := make([]T, 0, len(senders))
	for value := range outputChan {
		values = append(values, value)
	}
	return values
}

func manageLaunch[T any](outputChan chan<- T, senders []func(chan<- T), limit int) {
	guard := make(chan empty, limit) // set a limited number of "concurrent slot"
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(senders))
	for _, sender := range senders {
		guard <- empty{}     // take a concurrent slot (block until one is available)
		senderCopy := sender // avoid closure capture
		go func() {
			senderCopy(outputChan)
			<-guard // return the concurrent slot
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()  // all work is done
	close(outputChan) // no more sending
}
