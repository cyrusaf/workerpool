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

func TestWorkerPool(t *testing.T) {
	tt := []struct {
		name string
		desc string

		dropWhenFull bool
		numWorkers   int
		queueSize    int
		jobFunc      workerpool.Job
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
			jobFunc: func(ctx context.Context) {
				time.Sleep(10 * time.Millisecond)
			},
			jobsToQueue: 100,

			expectedJobsExecuted: 100,
		},
		{
			name:         "ExitOnContextCancel",
			desc:         "",
			dropWhenFull: false,
			numWorkers:   10,
			queueSize:    10,
			jobFunc: func(ctx context.Context) {
				<-ctx.Done()
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
				wp.Enqueue(func(ctx context.Context) {
					atomic.AddUint64(&jobsExecuted, 1)
					tc.jobFunc(ctx)
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

func TestWorkerPoolDropWhenFull(t *testing.T) {
	tc := struct {
		desc         string
		dropWhenFull bool
		numWorkers   int
		queueSize    int
		jobFunc      workerpool.Job
		jobsToQueue  int

		expectedJobsExecutedLessThan uint64
	}{
		desc:         "",
		dropWhenFull: true,
		numWorkers:   1,
		queueSize:    10,
		jobFunc: func(ctx context.Context) {
			time.Sleep(100 * time.Millisecond)
		},
		jobsToQueue: 100,

		expectedJobsExecutedLessThan: 50,
	}

	ctx := context.Background()
	wp := workerpool.WorkerPool{
		NumWorkers:   tc.numWorkers,
		QueueSize:    tc.queueSize,
		DropWhenFull: tc.dropWhenFull,
	}
	wp.Start(ctx)

	var jobsExecuted uint64 = 0
	for i := 0; i < tc.jobsToQueue; i++ {
		wp.Enqueue(func(ctx context.Context) {
			atomic.AddUint64(&jobsExecuted, 1)
			tc.jobFunc(ctx)
		})
	}

	wp.Shutdown()
	if atomic.LoadUint64(&jobsExecuted) >= tc.expectedJobsExecutedLessThan {
		t.Errorf("Expected less than %+v jobs executed, but %+v were executed instead\n", tc.expectedJobsExecutedLessThan, jobsExecuted)
	}
}

func TestWorkerPoolMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp := workerpool.WorkerPool{
		NumWorkers: 1,
		QueueSize:  5,
	}
	wp.Start(ctx)

	if activeWorkers := wp.ActiveWorkers(); activeWorkers != 0 {
		t.Errorf("Expected 0 active workers but got %d instead", activeWorkers)
	}
	if pendingJobs := wp.PendingJobs(); pendingJobs != 0 {
		t.Errorf("Expected 0 pending jobs but got %d instead", pendingJobs)
	}

	checkpoint := make(chan struct{})
	wp.Enqueue(func(ctx context.Context) {
		checkpoint <- struct{}{}
		<-ctx.Done()
	})

	<-checkpoint // Wait until job has been started
	if activeWorkers := wp.ActiveWorkers(); activeWorkers != 1 {
		t.Errorf("Expected 1 active worker but got %d instead", activeWorkers)
	}
	if pendingJobs := wp.PendingJobs(); pendingJobs != 0 {
		t.Errorf("Expected 0 pending jobs but got %d instead", pendingJobs)
	}

	// Fill up queue with jobs
	for i := 0; i < 5; i++ {
		wp.Enqueue(func(ctx context.Context) {
			<-ctx.Done()
		})
	}

	if pendingJobs := wp.PendingJobs(); pendingJobs != 5 {
		t.Errorf("Expected 5 pending jobs but got %d instead", pendingJobs)
	}

	cancel()

	// Wait for all work to be processed. A bit of a hack, could use channels for a cleaner
	// solution.
	time.Sleep(time.Second)

	if activeWorkers := wp.ActiveWorkers(); activeWorkers != 0 {
		t.Errorf("Expected 0 active workers but got %d instead", activeWorkers)
	}
	if pendingJobs := wp.PendingJobs(); pendingJobs != 0 {
		t.Errorf("Expected 0 pending jobs but got %d instead", pendingJobs)
	}
}
