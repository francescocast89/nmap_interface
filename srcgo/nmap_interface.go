package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrScanTimeout = errors.New("nmap scan timed out")
var ErrNmapNotInstalled = errors.New("nmap was not found")

// https://github.com/Ullaakut/nmap/blob/master/nmap.go
// Scanner represents an Nmap scanner.
type Scanner struct {
	cmd            *exec.Cmd
	args           []string
	path           string
	ctx            context.Context
	portFilter     func(Port) bool
	hostFilter     func(Host) bool
	stderr, stdout bufio.Scanner
}

type ScannerOption func(*Scanner)

func WithContext(ctx context.Context) ScannerOption {
	return func(s *Scanner) {
		s.ctx = ctx
	}
}

func WithPath(binaryPath string) ScannerOption {
	return func(s *Scanner) {
		s.path = binaryPath
	}
}

func WithCustomArguments(args ...string) ScannerOption {
	return func(s *Scanner) {
		s.args = append(s.args, args...)
	}
}
func WithTargets(targets ...string) ScannerOption {
	return func(s *Scanner) {
		s.args = append(s.args, targets...)
	}
}
func WithTargetExclusion(target string) ScannerOption {
	return func(s *Scanner) {
		s.args = append(s.args, "--exclude")
		s.args = append(s.args, target)
	}
}
func WithFilterPort(portFilter func(Port) bool) ScannerOption {
	return func(s *Scanner) {
		s.portFilter = portFilter
	}
}
func WithFilterHost(hostFilter func(Host) bool) ScannerOption {
	return func(s *Scanner) {
		s.hostFilter = hostFilter
	}
}

func WithPorts(ports ...string) ScannerOption {
	portList := strings.Join(ports, ",")

	return func(s *Scanner) {
		// Find if any port is set.
		var place int = -1
		for p, value := range s.args {
			if value == "-p" {
				place = p
				break
			}
		}
		// Add ports.
		if place >= 0 {
			portList = s.args[place+1] + "," + portList
			s.args[place+1] = portList
		} else {
			s.args = append(s.args, "-p")
			s.args = append(s.args, portList)
		}
	}
}

func WithPortExclusions(ports ...string) ScannerOption {
	portList := strings.Join(ports, ",")

	return func(s *Scanner) {
		s.args = append(s.args, "--exclude-ports")
		s.args = append(s.args, portList)
	}
}

/*** Timing and performance ***/

type Timing int16

const (
	TimingSlowest    Timing = 0
	TimingSneaky     Timing = 1
	TimingPolite     Timing = 2
	TimingNormal     Timing = 3
	TimingAggressive Timing = 4
	TimingFastest    Timing = 5
)

func WithTimingTemplate(timing Timing) ScannerOption {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-T%d", timing))
	}
}

var defaultArgs = []string{"-oX", "-", "--privileged"}

func NewScanner(options ...ScannerOption) (*Scanner, error) {

	s := &Scanner{}
	for _, option := range options {
		option(s)
	}

	if len(strings.TrimSpace(s.path)) == 0 {
		p, err := exec.LookPath("nmap")
		if err != nil {
			return nil, ErrNmapNotInstalled
		}
		s.path = p
	}
	if s.ctx == nil {
		s.ctx = context.Background()
	}
	return s, nil
}

func chooseHosts(result *Run, filter func(Host) bool) *Run {
	var filteredHosts []Host

	for _, host := range result.Hosts {
		if filter(host) {
			filteredHosts = append(filteredHosts, host)
		}
	}

	result.Hosts = filteredHosts

	return result
}

func choosePorts(result *Run, filter func(Port) bool) *Run {
	for idx := range result.Hosts {
		var filteredPorts []Port

		for _, port := range result.Hosts[idx].Ports {
			if filter(port) {
				filteredPorts = append(filteredPorts, port)
			}
		}
		result.Hosts[idx].Ports = filteredPorts
	}

	return result
}

func (s *Scanner) Run() (result *Run, warnings []string, err error) {
	var (
		stdout, stderr bytes.Buffer
	)
	args := append(s.args, defaultArgs...)
	s.cmd = exec.Command(s.path, args...)

	s.cmd.Stdout = &stdout
	s.cmd.Stderr = &stderr
	if err := s.cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("error during start: %s", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {

	case <-s.ctx.Done():
		_ = s.cmd.Process.Kill()
		return nil, warnings, s.ctx.Err()

	case <-done:
		if stderr.Len() > 0 {
			warnings = strings.Split(strings.Trim(stderr.String(), "\n"), "\n")
		}
		result, err := Parse(stdout.Bytes())
		if err != nil {
			return nil, warnings, fmt.Errorf("error during the xml parsing: %s", err)
		}
		if s.portFilter != nil {
			result = choosePorts(result, s.portFilter)
		}
		if s.hostFilter != nil {
			result = chooseHosts(result, s.hostFilter)
		}

		return result, warnings, nil
	}
}

func (s *Scanner) RunAsync() error {
	s.cmd = exec.Command(s.path, s.args...)

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("unable to get error output from asynchronous nmap run: %v", err)
	}

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("unable to get standard output from asynchronous nmap run: %v", err)
	}

	s.stdout = *bufio.NewScanner(stdout)
	s.stderr = *bufio.NewScanner(stderr)

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("error during start: %s", err)
	}

	go func() {
		<-s.ctx.Done()
		_ = s.cmd.Process.Kill()
	}()
	return nil
}

func NetworkScan(netAddr string, ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hostsList := make(map[string]string)

	s, err := NewScanner(WithCustomArguments("-PR", "-sn", "-n"), WithTargets(netAddr), WithContext(ctx))
	if err != nil {
		return nil, err
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return nil, fmt.Errorf("error during the scan process: %s", err)
		case context.Canceled:
			return nil, fmt.Errorf("scanner teminated: %s", err)
		case context.DeadlineExceeded:
			return nil, fmt.Errorf("scanner teminated: %s", err)
		}
	} else {
		if len(warnings) > 0 {
			fmt.Println("warnings: ", warnings)
		}
		for _, h := range result.Hosts {
			var (
				ipv4Addr, macAddr string
			)
			for _, a := range h.Addresses {
				if a.AddrType == "mac" {
					macAddr = a.Addr
				}
				if a.AddrType == "ipv4" {
					ipv4Addr = a.Addr
				}
			}
			if macAddr == "" {
				macAddr = fmt.Sprintf("DUMMY_%s", ipv4Addr)
			}
			hostsList[macAddr] = ipv4Addr
		}
	}
	return hostsList, nil
}

func PortScan(hostAddr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, err := NewScanner(WithCustomArguments("-sS", "-O", "-sV", "-p-", "--open"), WithTargets(hostAddr), WithContext(ctx))
	if err != nil {
		return err
	}
	result, warnings, err := s.Run()
	if err != nil {
		switch err {
		default:
			return fmt.Errorf("error during the scan process: %s", err)
		case context.Canceled:
			return fmt.Errorf("scanner teminated: %s", err)
		case context.DeadlineExceeded:
			return fmt.Errorf("scanner teminated: %s", err)
		}
	} else {
		if len(warnings) > 0 {
			fmt.Println("warnings: ", warnings)
		}
		for _, h := range result.Hosts {
			for _, p := range h.Ports {
				fmt.Printf("%0.d : %s \n", p.ID, p.Service)
			}
		}

	}
	return nil
}
