package main

import (
	"context"
	"fmt"
	"sync"
	"time"
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

type NetworkScanner struct {
	ctx     context.Context
	id      JobID
	netAddr string
}

type PortScanner struct {
	ctx  context.Context
	id   JobID
	host scannedHost
}

func NewNetworkScanner(ctx context.Context, id JobID, netAddr string) *NetworkScanner {
	return &NetworkScanner{ctx, id, netAddr}
}

func (ns *NetworkScanner) Id() JobID {
	return ns.id
}

func (ns *NetworkScanner) Execute() Result {
	ctx, cancel := context.WithCancel(ns.ctx)
	defer cancel()

	hl := make([]scannedHost, 0)

	s, err := NewScanner(WithCustomArguments("-PR", "-sn", "-n"), WithTargets(ns.netAddr), WithContext(ctx))
	if err != nil {
		return Result{ns.id, err, nil}
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return Result{ns.id, fmt.Errorf("error during the scan process: %s", err), hl}
		case context.Canceled:
			return Result{ns.id, fmt.Errorf("scanner teminated: %s", err), hl}
		case context.DeadlineExceeded:
			return Result{ns.id, fmt.Errorf("scanner teminated: %s", err), hl}
		}
	} else {
		if len(warnings) > 0 {
			fmt.Println("warnings: ", warnings)
		}
		for _, h := range result.Hosts {
			hl = append(hl, *NewScannedHost(h))
		}
	}
	return Result{ns.id, nil, hl}
}

func NewPortScanner(ctx context.Context, id JobID, host scannedHost) *PortScanner {
	return &PortScanner{ctx, id, host}
}

func (ps *PortScanner) Id() JobID {
	return ps.id
}
func (ps *PortScanner) Execute() Result {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s, err := NewScanner(WithCustomArguments("-sS", "-O", "-sV", "-p-", "--open"), WithTargets(ps.host.Ipv4Addr), WithContext(ctx))
	if err != nil {
		return Result{ps.id, err, nil}
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return Result{ps.id, fmt.Errorf("error during the scan process: %s", err), nil}
		case context.Canceled:
			return Result{ps.id, fmt.Errorf("scanner teminated: %s", err), nil}
		case context.DeadlineExceeded:
			return Result{ps.id, fmt.Errorf("scanner teminated: %s", err), nil}
		}
	} else {
		if len(warnings) > 0 {
			fmt.Println("warnings: ", warnings)
		}
		for _, h := range result.Hosts {
			ps.host.AddScannedOS(h)
			ps.host.AddScannedPort(h)
		}
	}
	return Result{ps.id, nil, ps.host}
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
