package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

//-------------------------------------------------------------------------------------------------

type NetworkScanner struct {
	ctx            context.Context
	id             JobID
	netAddr        string
	timingTemplate string
	scanTimeout    time.Duration
}

func NewNormalNetworkScanner(ctx context.Context, id JobID, netAddr string) *NetworkScanner {
	return &NetworkScanner{ctx, id, netAddr, "-T3", 1 * time.Hour}
}

func NewInsaneNetworkScanner(ctx context.Context, id JobID, netAddr string) *NetworkScanner {
	return &NetworkScanner{ctx, id, netAddr, "-T5", 3 * time.Hour}
}

func (ns *NetworkScanner) Id() JobID {
	return ns.id
}

func (ns *NetworkScanner) Execute() Result {
	ctx, cancel := context.WithTimeout(ns.ctx, ns.scanTimeout)
	defer cancel()

	hl := make([]scannedHost, 0)

	s, err := NewScanner(WithCustomArguments("-PR", "-sn", "-n", ns.timingTemplate), WithTargets(ns.netAddr), WithContext(ctx))
	if err != nil {
		return Result{ns.id, err, nil}
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return Result{ns.id, fmt.Errorf("error during the scan process: %s", err), ns.netAddr}
		case context.Canceled:
			return Result{ns.id, fmt.Errorf("scanner teminated: %s", err), ns.netAddr}
		case context.DeadlineExceeded:
			return Result{ns.id, fmt.Errorf("scanner teminated: %s", err), ns.netAddr}
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

func NewNetworkScannerPool(ctx context.Context, numberOfWorkers int, netList []string) []scannedHost {
	fmt.Println("start the emitter")
	results := make([]Result, 0)
	scannedHostsList := make([]scannedHost, 0)
	slowNetworksList := make([]string, 0)
	wp := NewWorkerPool(numberOfWorkers, ctx)
	go wp.Run(ctx)
	go wp.Collector(ctx, &results)

	go func() {
		for _, i := range netList {
			wp.jobs <- NewNormalNetworkScanner(ctx, JobID(i), i)
		}
		close(wp.jobs)
	}()
	<-wp.Done
	close(wp.results)
	<-wp.Done
	for _, i := range results {
		if i.Err != nil {
			slowNetworksList = append(slowNetworksList, ToNetworkAddr(i.Value))
			continue
		}
		scannedHostsList = append(scannedHostsList, ToScannedHostList(i.Value)...)
	}
	if len(slowNetworksList) > 0 {
		fmt.Println("start to scan slow networks")
		results := make([]Result, 0)
		wp := NewWorkerPool(numberOfWorkers, ctx)
		go wp.Run(ctx)
		go wp.Collector(ctx, &results)

		go func() {
			for _, i := range slowNetworksList {

				wp.jobs <- NewInsaneNetworkScanner(ctx, JobID(i), i)
			}
			close(wp.jobs)
		}()
		<-wp.Done
		close(wp.results)
		<-wp.Done
		for _, i := range results {
			if i.Err != nil {
				fmt.Println(fmt.Errorf("error during scan of network %s:%s", ToNetworkAddr(i.Value), i.Err))
				continue
			}
			scannedHostsList = append(scannedHostsList, ToScannedHostList(i.Value)...)
		}
	}
	return scannedHostsList
}

func ToNetworkAddr(x interface{}) string {
	v, ok := x.(string)
	if !ok {
		return ""
	}
	return v
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

//-------------------------------------------------------------------------------------------------

type PortScanner struct {
	ctx            context.Context
	id             JobID
	host           scannedHost
	timingTemplate string
	scanTimeout    time.Duration
}

func NewNormalPortScanner(ctx context.Context, id JobID, host scannedHost) *PortScanner {
	return &PortScanner{ctx, id, host, "-T3", 30 * time.Second}
}

func NewInsanePortScanner(ctx context.Context, id JobID, host scannedHost) *PortScanner {
	return &PortScanner{ctx, id, host, "-T5", 30 * time.Minute}
}

func (ps *PortScanner) Id() JobID {
	return ps.id
}
func (ps *PortScanner) Execute() Result {
	ctx, cancel := context.WithTimeout(ps.ctx, ps.scanTimeout)
	defer cancel()

	s, err := NewScanner(WithCustomArguments("-sS", "-O", "-sV", "-p-", "--open", ps.timingTemplate), WithTargets(ps.host.Ipv4Addr), WithContext(ctx))
	if err != nil {
		return Result{ps.id, err, nil}
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return Result{ps.id, fmt.Errorf("error during the scan process: %s", err), ps.host}
		case context.Canceled:
			return Result{ps.id, fmt.Errorf("scanner teminated: %s", err), ps.host}
		case context.DeadlineExceeded:
			return Result{ps.id, fmt.Errorf("scanner teminated: %s", err), ps.host}
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

func NewPortScannerPool(ctx context.Context, numberOfWorkers int, hostList []scannedHost) []scannedHost {
	fmt.Println("start the emitter")
	results := make([]Result, 0)
	scannedHostsList := make([]scannedHost, 0)
	slowHostsList := make([]scannedHost, 0)
	wp := NewWorkerPool(numberOfWorkers, ctx)
	go wp.Run(ctx)
	go wp.Collector(ctx, &results)

	go func() {
		for idx, i := range hostList {
			wp.jobs <- NewNormalPortScanner(ctx, JobID(fmt.Sprintf("%s_%d", i.Ipv4Addr, idx)), i)
		}
		close(wp.jobs)
	}()
	<-wp.Done
	close(wp.results)
	<-wp.Done
	for _, i := range results {
		if i.Err != nil {
			slowHostsList = append(slowHostsList, ToScannedHost(i.Value))
			continue
		}
		scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
	}
	if len(scannedHostsList) > 0 {
		results := make([]Result, 0)
		fmt.Println("start to scan slow hosts")
		wp := NewWorkerPool(numberOfWorkers, ctx)
		go wp.Run(ctx)
		go wp.Collector(ctx, &results)

		go func() {
			for idx, i := range slowHostsList {
				wp.jobs <- NewInsanePortScanner(ctx, JobID(fmt.Sprintf("%s_%d", i.Ipv4Addr, idx)), i)
			}
			close(wp.jobs)
		}()
		<-wp.Done
		close(wp.results)
		<-wp.Done
		for _, i := range results {
			if i.Err != nil {
				fmt.Println(fmt.Errorf("error during scan of host %s:%s", ToScannedHost(i.Value).Ipv4Addr, i.Err))
				continue
			}
			scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
		}
	}
	return scannedHostsList
}

//-------------------------------------------------------------------------------------------------

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

	for i := 1; i <= wp.workersCount; i++ {
		go wp.worker(i, ctx, &wg)
		wg.Add(1)
	}
	wg.Wait()

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

func (wp WorkerPool) Collector(ctx context.Context, results *[]Result) {
	fmt.Println("start to collect")
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
