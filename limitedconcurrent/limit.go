package limitedconcurrent

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
	guard := make(chan empty, limit) // initialize a limited number of "concurrent slot"
	done := func() {
		<-guard // release a concurrent slot
	}

	for _, sender := range senders {
		guard <- empty{}     // take a concurrent slot (block until one is available)
		senderCopy := sender // avoid closure capture
		go func() {
			defer done() // use defer because sender can panic
			senderCopy(outputChan)
		}()
	}

	// fill all available slot
	for i := 0; i < limit; i++ {
		guard <- empty{}
	}
	close(outputChan) // all work is done, no more sending
}
