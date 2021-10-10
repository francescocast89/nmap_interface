package main

import (
	"context"
	"fmt"
	"sync"
)

type JobID string

type ExecutionFn func(ctx context.Context) (interface{}, error)

type slot struct{}

type Job struct {
	Id     JobID
	ExecFn ExecutionFn
}
type Result struct {
	Value interface{}
	Err   error
	Id    JobID
}

// type WorkerPool struct {
// 	workersCount int
// 	jobs         chan Job
// 	results      chan Result
// 	slots        chan slot
// 	Done         chan bool
// }
type WorkerPool struct {
	workersCount int
	jobs         chan Job
	results      chan Result
}

func (j Job) execute(ctx context.Context) Result {
	value, err := j.ExecFn(ctx)
	if err != nil {
		return Result{
			Err: err,
			Id:  j.Id,
		}
	}
	return Result{
		Value: value,
		Id:    j.Id,
	}
}
func NewWorkerPool(wcount int, jcount int, ctx context.Context) WorkerPool {
	fmt.Println("new worker pool")
	wp := WorkerPool{
		workersCount: wcount,
		jobs:         make(chan Job, wcount),
		results:      make(chan Result, jcount),
	}
	return wp
}

func (wp WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 1; i <= wp.workersCount; i++ {
		go wp.worker(i, ctx, &wg)
		wg.Add(1)
	}

	wg.Wait()
}

func (wp WorkerPool) worker(id int, ctx context.Context, wg *sync.WaitGroup) {
	fmt.Println("start worker")
	defer wg.Done()
	for {
		select {
		case j, ok := <-wp.jobs:
			if !ok {
				fmt.Println("worker terminated")
				return
			}
			fmt.Printf("job %s started by worker %0.d \n", j.Id, id)
			select {
			case wp.results <- j.execute(ctx):
				fmt.Printf("job %s executed by worker %0.d \n", j.Id, id)
			case <-ctx.Done():
				fmt.Println("worker terminated by context")
				return
			}
		case <-ctx.Done():
			fmt.Println("worker terminated by context")
			return
		}
	}
}

// func NewWorkerPool(wcount int) WorkerPool {
// 	fmt.Println("new worker pool")
// 	wp := WorkerPool{
// 		workersCount: wcount,
// 		jobs:         make(chan Job),
// 		results:      make(chan Result),
// 		slots:        make(chan slot, wcount),
// 		Done:         make(chan bool),
// 	}
// 	return wp
// }
// func (wp WorkerPool) StartDispatcher(ctx context.Context) {
// 	fmt.Println("start dispatcher")
// 	for {
// 		select {
// 		case j := <-wp.jobs:
// 			fmt.Println("wp.job: ", j.Id)
// 			fmt.Printf("%0.d / %0.d \n", len(wp.slots), cap(wp.slots))
// 			select {
// 			case wp.slots <- slot{}:
// 				go func() { wp.results <- j.execute() }()
// 				fmt.Println("job inserted: ", j.Id)
// 			case <-ctx.Done():
// 				fmt.Println("Dispatcher terminated by context")
// 				return
// 			}
// 		case <-ctx.Done():
// 			fmt.Println("Dispatcher terminated by context")
// 			return
// 		}
// 	}
// }
// func (wp WorkerPool) StartCollector(ctx context.Context) {
// 	fmt.Println("start collector")
// 	for {
// 		select {
// 		case j := <-wp.results:
// 			select {
// 			case <-wp.slots:
// 				fmt.Println(j.Id, j.Value)
// 			case <-ctx.Done():
// 				fmt.Println("Collector terminated by context")
// 				return
// 			}
// 		case <-ctx.Done():
// 			fmt.Println("Collector terminated by context")
// 			return
// 		}
// 	}
// }

func (wp WorkerPool) StartCollector(ctx context.Context) {
	for {
		select {
		case j := <-wp.results:
			fmt.Println(j.Id, j.Value)
		case <-ctx.Done():
			fmt.Println("collector terminated by context")
			return
		}
	}
}
func (wp WorkerPool) addJob(id string, f ExecutionFn) {
	j := Job{
		Id:     JobID(id),
		ExecFn: f,
	}
	wp.jobs <- j
}
