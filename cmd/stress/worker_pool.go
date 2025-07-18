package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job func(ctx context.Context) error

type WorkerPool struct {
	workers    int
	jobTimeout time.Duration
	jobs       chan Job
	wg         sync.WaitGroup
	errs       chan error
	closed     chan struct{}
}

func NewWorkerPool(workers int, jobTimeout time.Duration) *WorkerPool {
	return &WorkerPool{
		workers:    workers,
		jobTimeout: jobTimeout,
		jobs:       make(chan Job, workers),
		errs:       make(chan error, workers),
		closed:     make(chan struct{}),
	}
}

func (p *WorkerPool) Start(ctx context.Context) <-chan error {
	for i := 0; i < p.workers; i++ {
		go func(workerID int) {
			for job := range p.jobs {
				jctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), p.jobTimeout)
				if err := job(jctx); err != nil {
					p.errs <- fmt.Errorf("worker %d job failed: %w", workerID, err)
				}
				cancel()
				p.wg.Done()
			}
		}(i)
	}

	return p.errs
}

func (p *WorkerPool) QueueJob(job Job) {
	select {
	case <-p.closed:
		return // nop if pool is closed
	default:
	}

	p.wg.Add(1)
	select {
	case p.jobs <- job:
	case <-p.closed:
		p.wg.Done() // job not scheduled so undo add
	}
}

func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

func (p *WorkerPool) Close() {
	close(p.closed)
	close(p.jobs)
	p.wg.Wait()
	close(p.errs)
}
