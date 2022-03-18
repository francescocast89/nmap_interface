package main

import (
	"context"
	"fmt"
	"time"
)

const NetworkScannerProgressRefreshSeconds = 10

type NetworkScanner struct {
	ctx                 context.Context
	id                  JobID
	netAddr             string
	timingTemplate      Timing
	scanTimeout         time.Duration
	liveProgressChannel chan float32
}

func NewNormalNetworkScanner(ctx context.Context, id JobID, netAddr string, p chan float32) *NetworkScanner {
	return &NetworkScanner{ctx, id, netAddr, 3, 1 * time.Hour, p}
}

func NewInsaneNetworkScanner(ctx context.Context, id JobID, netAddr string, p chan float32) *NetworkScanner {
	return &NetworkScanner{ctx, id, netAddr, 5, 3 * time.Hour, p}
}

func (ns *NetworkScanner) Id() JobID {
	return ns.id
}

func (ns *NetworkScanner) Execute() Result {
	ctx, cancel := context.WithTimeout(ns.ctx, ns.scanTimeout)
	defer cancel()

	hl := make([]scannedHost, 0)

	s, err := NewScanner(WithCustomArguments("-PR", "-sn", "-n"), WithTimingTemplate(ns.timingTemplate), WithTargets(ns.netAddr), WithContext(ctx))
	if err != nil {
		return Result{ns.id, err, nil}
	}

	var result *Run
	var warnings []string
	if ns.liveProgressChannel == nil {
		result, warnings, err = s.Run()
	} else {
		result, warnings, err = s.RunWithProgress(ns.liveProgressChannel, NetworkScannerProgressRefreshSeconds)
	}
	if err != nil {
		switch err {
		default:
			return Result{ns.id, fmt.Errorf("error during the scan process: %s", err), ns.netAddr}
		case context.Canceled:
			return Result{ns.id, fmt.Errorf("scanner teminated: %s", err), ns.netAddr}
		case context.DeadlineExceeded:
			return Result{ns.id, err, ns.netAddr}
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
	results := make([]Result, 0)
	scannedHostsList := make([]scannedHost, 0)
	slowNetworksList := make([]string, 0)
	// first pool try to scan all the networks in the list, if the timeout expire
	// the netwkork addr is appended into a slowNetork list, these networks are scanned by a new pool when the
	// first one has terminated
	wp := NewWorkerPool(numberOfWorkers, ctx)
	go wp.run(ctx)
	go wp.collector(ctx, &results)
	go func() {
		for _, i := range netList {
			wp.jobs <- NewNormalNetworkScanner(ctx, JobID(i), i, nil)
		}
		close(wp.jobs)
	}()
	<-wp.WorkersDone
	close(wp.results)
	<-wp.CollectorDone
	for _, i := range results {
		if i.Err != nil {
			if i.Err == context.DeadlineExceeded {
				// if the error is the timeout of the context, append the host to the slowNetworks list
				slowNetworksList = append(slowNetworksList, ToNetworkAddr(i.Value))
			}
			continue
		}
		scannedHostsList = append(scannedHostsList, ToScannedHostList(i.Value)...)
	}
	// second pool encharged for scanning slow networks
	if len(slowNetworksList) > 0 {
		fmt.Println("start to scan slow networks")
		results := make([]Result, 0)
		wp := NewWorkerPool(numberOfWorkers, ctx)
		go wp.run(ctx)
		go wp.collector(ctx, &results)

		go func() {
			for _, i := range slowNetworksList {

				wp.jobs <- NewInsaneNetworkScanner(ctx, JobID(i), i, nil)
			}
			close(wp.jobs)
		}()
		<-wp.WorkersDone
		close(wp.results)
		<-wp.CollectorDone
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
