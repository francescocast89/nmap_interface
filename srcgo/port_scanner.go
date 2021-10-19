package main

import (
	"context"
	"fmt"
	"time"
)

type PortScanner struct {
	ctx  context.Context
	id   JobID
	host scannedHost
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
