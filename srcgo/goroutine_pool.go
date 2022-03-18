package main

import (
	"context"
	"fmt"
	"sync"
)

type JobID string

type Job interface {
	Id() JobID
	Execute() Result
}

type Result struct {
	Id    JobID
	Err   error
	Value interface{}
}

type WorkerPool struct {
	workersCount  int
	jobs          chan Job
	results       chan Result
	WorkersDone   chan bool
	CollectorDone chan bool
}

func NewWorkerPool(wcount int, ctx context.Context) WorkerPool {
	wp := WorkerPool{
		workersCount:  wcount,
		jobs:          make(chan Job, wcount),
		results:       make(chan Result),
		WorkersDone:   make(chan bool),
		CollectorDone: make(chan bool),
	}
	return wp
}

func (wp WorkerPool) run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 1; i <= wp.workersCount; i++ {
		go wp.worker(i, ctx, &wg)
		wg.Add(1)
	}
	wg.Wait()

	wp.WorkersDone <- true
}

func (wp WorkerPool) worker(id int, ctx context.Context, wg *sync.WaitGroup) {
	fmt.Printf("start worker %0.d\n", id)
	defer wg.Done()
	for {
		select {
		case j, ok := <-wp.jobs:
			if !ok {
				fmt.Printf("worker %0.d terminated\n", id)
				return
			}
			fmt.Printf("job %s assigned to worker %0.d\n", j.Id(), id)
			select {
			case wp.results <- j.Execute():
			case <-ctx.Done():
				fmt.Printf("worker %0.d terminated by context\n", id)
				return
			}
		case <-ctx.Done():
			fmt.Printf("worker %0.d terminated by context\n", id)
			return
		}
	}
}

func (wp WorkerPool) collector(ctx context.Context, results *[]Result) {
	fmt.Println("start to collect")
	for {
		select {
		case j, ok := <-wp.results:
			if !ok {
				fmt.Printf("collector terminated\n")
				wp.CollectorDone <- true
				return
			}
			*results = append(*results, j)
		case <-ctx.Done():
			fmt.Println("collector terminated by context")
			return
		}
	}
}
