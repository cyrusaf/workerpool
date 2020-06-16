package workerpool

import (
	"context"
	"sync"
	"sync/atomic"
)

// WorkerPool spins up multiple workers to process jobs in the background. A WorkerPool
// processes a queue of jobs. Use Enqueue(j Job) to add a job to the queue. It is not safe
// to change the values of WorkerPool while it is running.
type WorkerPool struct {

	// NumWorkers determines how many workers are spun up to process jobs.
	NumWorkers int

	// QueueSize determines the size of the job queue. Once the job queue is full,
	// enqueuing jobs will block unless DropWhenFull is enabled.
	QueueSize int

	// DropWhenFull will drop jobs instead of blocking if the job queue is full when set
	// to true.
	DropWhenFull bool

	// OnJobDropped is called each time a job is dropped. Can be used for adding metrics
	// or logging around dropped jobs.
	OnJobDropped func() // TODO: add context as arg

	wg            sync.WaitGroup
	jobs          chan func(ctx context.Context)
	running       bool
	activeWorkers int64
}

// Start kicks off a specified number of workers with a specified size for the job queue.
func (w *WorkerPool) Start(ctx context.Context) {
	if w.running {
		panic("Called Start() on already running WorkerPool")
	}
	if w.NumWorkers == 0 {
		panic("Must have more than 0 workers")
	}

	w.running = true
	w.jobs = make(chan func(ctx context.Context), w.QueueSize)
	for i := 0; i < w.NumWorkers; i++ {
		w.newWorker(ctx)
	}
}

// Shutdown will gracefully shutdown the workerpool and block until all jobs in the queue
// have been processed. In order to hard shutdown the workerpool, cancel the context passed
// into Start(ctx context.Context).
func (w *WorkerPool) Shutdown() {
	close(w.jobs)
	w.wg.Wait()
	w.running = false
}

// Enqueue adds a specified job onto the job queue.
func (w *WorkerPool) Enqueue(job func(ctx context.Context)) {
	switch w.DropWhenFull {
	case true:
		select {
		case w.jobs <- job:
		default:
			if w.OnJobDropped != nil {
				w.OnJobDropped()
			}
		}
	case false:
		w.jobs <- job
	}
}

// PendingJobs returns the number of jobs waiting to be processed.
func (w *WorkerPool) PendingJobs() int {
	return len(w.jobs)
}

// ActiveWorkers returns the number of workers currently processing a job.
func (w *WorkerPool) ActiveWorkers() int {
	return int(atomic.LoadInt64(&w.activeWorkers))
}

func (w *WorkerPool) newWorker(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for job := range w.jobs { // TODO: use context here?
			atomic.AddInt64(&w.activeWorkers, 1)
			job(ctx)
			atomic.AddInt64(&w.activeWorkers, -1)
		}
	}()
}
