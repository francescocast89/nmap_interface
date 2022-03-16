package main

import (
	"context"
	"fmt"
	"time"
)

type PortScanner struct {
	ctx            context.Context
	id             JobID
	host           scannedHost
	timingTemplate Timing
	scanTimeout    time.Duration
}

func NewNormalPortScanner(ctx context.Context, id JobID, host scannedHost) *PortScanner {
	return &PortScanner{ctx, id, host, 3, 30 * time.Second}
}

func NewInsanePortScanner(ctx context.Context, id JobID, host scannedHost) *PortScanner {
	return &PortScanner{ctx, id, host, 5, 30 * time.Minute}
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
	results := make([]Result, 0)
	scannedHostsList := make([]scannedHost, 0)
	slowHostsList := make([]scannedHost, 0)
	// first pool try to scan all the hosts in the list, if the timeout expire
	// the host is appended into a slowHosts list, these hosts are scanned by a new pool when the
	// first one has terminated
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
			if i.Err == context.DeadlineExceeded {
				// if the error is the timeout of the context, append the host to the slowHosts list
				slowHostsList = append(slowHostsList, ToScannedHost(i.Value))
			}
			continue
		}
		scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
	}
	// second pool encharged for scanning slow hosts
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
				// if there is an error during this second scan (can also be a timeout error), log the error and continue
				fmt.Println(fmt.Errorf("error during scan of host %s:%s", ToScannedHost(i.Value).Ipv4Addr, i.Err))
				continue
			}
			scannedHostsList = append(scannedHostsList, ToScannedHost(i.Value))
		}
	}
	return scannedHostsList
}
