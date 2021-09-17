package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Streamer constantly streams the stdout.
type Streamer interface {
	Write(d []byte) (int, error)
	Bytes() []byte
}

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

var defaultArgs = []string{"-oX", "-"}

func NewScanner(args []string) (*Scanner, error) {

	s := &Scanner{}

	if len(args) == 0 {
		return nil, fmt.Errorf("provided 0 arguments")
	}
	s.args = append(args, defaultArgs...)
	trimmedToolPath := strings.TrimSpace(s.path)
	if len(trimmedToolPath) == 0 {
		p, err := exec.LookPath("nmap")
		if err != nil {
			return nil, fmt.Errorf("unable to find %s: %v", s.path, err)
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
	s.cmd = exec.Command(s.path, s.args...)
	s.cmd.Stdout = &stdout
	s.cmd.Stderr = &stderr

	if err := s.cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("Run(): error during start: ", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {

	case <-s.ctx.Done():
		_ = s.cmd.Process.Kill()
		return nil, nil, nil

	case <-done:
		if stderr.Len() > 0 {
			warnings = strings.Split(strings.Trim(stderr.String(), "\n"), "\n")
		}
		result, err := Parse(stdout.Bytes())
		if err != nil {
			return nil, warnings, fmt.Errorf("error during parse: ", err)
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

func main() {
	s, err := NewScanner([]string{"-PR", "-sn", "-n", "192.168.73.0/24"})
	if err != nil {
		fmt.Println(err)
	}
	result, warnings, err := s.Run()
	if err != nil {
		fmt.Println("error:", err)
	}
	if len(warnings) > 0 {
		fmt.Println("warnings: ", warnings)
	}
	for _, h := range result.Hosts {
		fmt.Println(h.Addresses)
	}

}
