package workerpool_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cyrusaf/workerpool"
)

func ExampleWorkerPool() {
	ctx := context.Background()
	wp := workerpool.WorkerPool{
		NumWorkers: 2,
		QueueSize:  10,
	}
	wp.Start(ctx) // Create worker goroutines and begin processing jobs

	var counter uint64 = 0 // Create counter to keep track of jobs run
	for i := 0; i < 5; i++ {
		// Enqueue job onto workerpool's job queue
		wp.Enqueue(func(ctx context.Context) error {
			atomic.AddUint64(&counter, 1) // Increment counter
			return nil
		})
	}
	// Gracefully shutdown workerpool, waiting for all queued jobs to finish
	wp.Shutdown()

	fmt.Println(counter)
	// Output: 5
}

func TestWorkerPool(t *testing.T) {
	tt := []struct {
		name string
		desc string

		dropWhenFull bool
		numWorkers   int
		queueSize    int
		jobFunc      func(ctx context.Context) error
		jobsToQueue  int
		hardShutdown bool

		expectedJobsExecuted uint64
	}{
		{
			name:         "BaseCase",
			desc:         "",
			dropWhenFull: false,
			numWorkers:   10,
			queueSize:    10,
			jobFunc: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			jobsToQueue: 100,

			expectedJobsExecuted: 100,
		},
		{
			name:         "DropWhenFull",
			desc:         "",
			dropWhenFull: true,
			numWorkers:   1,
			queueSize:    9,
			jobFunc: func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			jobsToQueue: 100,

			expectedJobsExecuted: 10,
		},
		{
			name:         "ExitOnContextCancel",
			desc:         "",
			dropWhenFull: false,
			numWorkers:   10,
			queueSize:    10,
			jobFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			jobsToQueue:  10,
			hardShutdown: true,

			expectedJobsExecuted: 10,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wp := workerpool.WorkerPool{
				NumWorkers:   tc.numWorkers,
				QueueSize:    tc.queueSize,
				DropWhenFull: tc.dropWhenFull,
			}
			wp.Start(ctx)

			var jobsExecuted uint64 = 0
			for i := 0; i < tc.jobsToQueue; i++ {
				wp.Enqueue(func(ctx context.Context) error {
					atomic.AddUint64(&jobsExecuted, 1)
					return tc.jobFunc(ctx)
				})
			}
			if tc.hardShutdown {
				cancel()
				fmt.Println("cancelled context")
			}
			wp.Shutdown()
			if atomic.LoadUint64(&jobsExecuted) != tc.expectedJobsExecuted {
				t.Errorf("Expected %+v jobs executed, but %+v were executed instead\n", tc.expectedJobsExecuted, jobsExecuted)
			}
		})
	}
}