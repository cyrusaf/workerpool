[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/cyrusaf/workerpool)
[![Build Status](https://travis-ci.org/cyrusaf/workerpool.svg?branch=master)](https://travis-ci.org/cyrusaf/workerpool)

Package workerpool provides a generic workerpool implementation with basic features
such as graceful shutdown and optionally dropping jobs instead of blocking when the queue
is full. See the [godoc](https://godoc.org/github.com/cyrusaf/workerpool) page for full documenation.

## Example
```golang
func main() {
	ctx := context.Background()
	wp := workerpool.WorkerPool{
		NumWorkers: 2,
		QueueSize:  10,
	}
	wp.Start(ctx) // Spawn worker goroutines and begin processing jobs

	var counter uint64 = 0 // Create counter to keep track of jobs run
	for i := 0; i < 5; i++ {
		// Enqueue job onto workerpool's job queue
		wp.Enqueue(func(ctx context.Context) {
			atomic.AddUint64(&counter, 1) // Increment counter
		})
	}
	// Gracefully shutdown workerpool, waiting for all queued jobs to finish
	wp.Shutdown()

	fmt.Println(counter)
	// Output: 5
}
```