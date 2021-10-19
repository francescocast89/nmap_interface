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
	workersCount int
	jobs         chan Job
	results      chan Result
	Done         chan bool
}

func ToMap(x interface{}) map[string]string {
	res := map[string]string{}
	v, ok := x.(map[string]string)
	if !ok {
		return res
	}
	for k, v := range v {
		res[k] = v
	}
	return res
}

func ToScannedHost(x interface{}) scannedHost {
	v, ok := x.(scannedHost)
	if !ok {
		return scannedHost{}
	}
	return v
}

func ToScannedHostList(x interface{}) []scannedHost {
	res := []scannedHost{}
	v, ok := x.([]scannedHost)
	if !ok {
		return res
	}
	res = append(res, v...)
	return res
}

func NewWorkerPool(wcount int, ctx context.Context) WorkerPool {
	wp := WorkerPool{
		workersCount: wcount,
		jobs:         make(chan Job, wcount),
		results:      make(chan Result),
		Done:         make(chan bool),
	}
	return wp
}

func (wp WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	// var cwg sync.WaitGroup

	// go wp.StartCollector(ctx, results, &cwg)
	// cwg.Add(1)

	for i := 1; i <= wp.workersCount; i++ {
		go wp.worker(i, ctx, &wg)
		wg.Add(1)
	}
	wg.Wait()
	// close(wp.results)
	// cwg.Wait()
	wp.Done <- true
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

func (wp WorkerPool) StartCollector(ctx context.Context, results *[]Result) {
	fmt.Println("start collector")
	for {
		select {
		case j, ok := <-wp.results:
			if !ok {
				fmt.Printf("collector terminated\n")
				wp.Done <- true
				return
			}
			*results = append(*results, j)
		case <-ctx.Done():
			fmt.Println("collector terminated by context")
			return
		}
	}
}

// func (wp WorkerPool) addJob(id string, f ExecutionFn) {
// 	j := Job{
// 		Id:     JobID(id),
// 		ExecFn: f,
// 	}
// 	wp.jobs <- j
// }
