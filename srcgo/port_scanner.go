package main

import (
	"context"
	"fmt"
	"time"
)

const PortScanScannerProgressRefreshSeconds = 10

type PortScanner struct {
	ctx                 context.Context
	id                  JobID
	host                scannedHost
	timingTemplate      Timing
	scanTimeout         time.Duration
	liveProgressChannel chan float32
}

func NewNormalPortScanner(ctx context.Context, id JobID, host scannedHost, p chan float32) *PortScanner {
	return &PortScanner{ctx, id, host, 3, 30 * time.Second, p}
}

func NewInsanePortScanner(ctx context.Context, id JobID, host scannedHost, p chan float32) *PortScanner {
	return &PortScanner{ctx, id, host, 5, 30 * time.Minute, p}
}

func (ps *PortScanner) Id() JobID {
	return ps.id
}
func (ps *PortScanner) Execute() Result {
	ctx, cancel := context.WithTimeout(ps.ctx, ps.scanTimeout)
	defer cancel()

	s, err := NewScanner(WithCustomArguments("-sS", "-O", "-sV", "-p-", "--open"), WithTimingTemplate(ps.timingTemplate), WithTargets(ps.host.Ipv4Addr), WithContext(ctx))
	if err != nil {
		return Result{ps.id, err, nil}
	}

	var result *Run
	var warnings []string
	if ps.liveProgressChannel == nil {
		result, warnings, err = s.Run()
	} else {
		result, warnings, err = s.RunWithProgress(ps.liveProgressChannel, PortScanScannerProgressRefreshSeconds)
	}

	if err != nil {
		switch err {
		default:
			return Result{ps.id, fmt.Errorf("error during the scan process: %s", err), ps.host}
		case context.Canceled:
			return Result{ps.id, fmt.Errorf("scanner teminated: %s", err), ps.host}
		case context.DeadlineExceeded:
			return Result{ps.id, err, ps.host}
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
	innerCtx, innerCtxCanc := context.WithCancel(ctx)
	defer innerCtxCanc()
	results := make([]Result, 0)
	scannedHostsList := make([]scannedHost, 0)
	slowHostsList := make([]scannedHost, 0)
	// first pool try to scan all the hosts in the list, if the timeout expire
	// the host is appended into a slowHosts list, these hosts are scanned by a new pool when the
	// first one has terminated
	wp := NewWorkerPool(numberOfWorkers, ctx)
	go wp.run(ctx)
	go wp.collector(ctx, &results)
	lpReader := NewLiveProgressReader(ctx, 10*time.Second)

	go func() {
		for _, i := range hostList {
			ch := make(chan float32)
			wp.jobs <- NewNormalPortScanner(ctx, JobID(fmt.Sprintf("%s", i.Ipv4Addr)), i, ch)
			lpReader.AddJobToLiveProgressReader(fmt.Sprintf("%s", i.Ipv4Addr), ch)
		}
		close(wp.jobs)
	}()
	go lpReader.PrintCurrentProgress(innerCtx, 10*time.Second)
	<-wp.WorkersDone
	close(wp.results)
	<-wp.CollectorDone
	for _, i := range results {
		if i.Err != nil {
			fmt.Println(i.Err)
			if i.Err == context.DeadlineExceeded {
				// if the error is the timeout of the context, append the host to the slowHosts list
				slowHostsList = append(slowHostsList, ToScannedHost(i.Value))
			}
			continue
		}
		scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
	}
	// second pool encharged for scanning slow hosts
	if len(slowHostsList) > 0 {
		results := make([]Result, 0)
		fmt.Println("start to scan slow hosts")
		wp := NewWorkerPool(numberOfWorkers, ctx)
		go wp.run(ctx)
		go wp.collector(ctx, &results)
		lpReader := NewLiveProgressReader(ctx, 1*time.Second)
		go func() {
			for idx, i := range slowHostsList {
				ch := make(chan float32)
				wp.jobs <- NewInsanePortScanner(ctx, JobID(fmt.Sprintf("%s_%d", i.Ipv4Addr, idx)), i, ch)
				lpReader.AddJobToLiveProgressReader(fmt.Sprintf("%s", i.Ipv4Addr), ch)
			}
			close(wp.jobs)
		}()
		go lpReader.PrintCurrentProgress(innerCtx, 10*time.Second)
		<-wp.WorkersDone
		close(wp.results)
		<-wp.CollectorDone
		for _, i := range results {
			if i.Err != nil {
				// if there is an error during this second scan (can also be a timeout error), log the error and continue
				fmt.Println(fmt.Errorf("error during scan of host %s:%s", ToScannedHost(i.Value).Ipv4Addr, i.Err))
				continue
			}
			scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
		}
	}
	return scannedHostsList
}
