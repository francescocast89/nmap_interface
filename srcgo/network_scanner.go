package main

import (
	"context"
	"fmt"
)

type NetworkScanner struct {
	ctx     context.Context
	id      JobID
	netAddr string
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
