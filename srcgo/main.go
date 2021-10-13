package main

import (
	"context"
	"fmt"
)

type scannedPort struct {
	Id      uint16
	Service string
}

type scannedOS struct {
	Name     string
	Accuracy int
}

type scannedHost struct {
	MacAddr  string
	Ipv4Addr string
	os       []scannedOS
	ports    []scannedPort
}

func NewScannedHost(h Host) *scannedHost {
	sh := &scannedHost{}
	for _, a := range h.Addresses {
		if a.AddrType == "mac" {
			sh.MacAddr = a.Addr
		}
		if a.AddrType == "ipv4" {
			sh.Ipv4Addr = a.Addr
		}
	}
	if sh.MacAddr == "" {
		sh.MacAddr = fmt.Sprintf("DUMMY_%s", sh.Ipv4Addr)
	}
	return sh
}

func (sh *scannedHost) AddScannedPort(h Host) {
	for _, p := range h.Ports {
		//fmt.Printf("%0.d : %s \n", p.ID, p.Service.Name)
		sh.ports = append(sh.ports, scannedPort{p.ID, p.Service.Name})
	}

}

func (sh *scannedHost) AddScannedOS(h Host) {
	for _, o := range h.OS.Matches {
		//fmt.Printf("%s, %0.d \n", o.Name, o.Accuracy)
		sh.os = append(sh.os, scannedOS{o.Name, o.Accuracy})
	}

}

func main() {
	// networksList := []string{"192.168.1.29/32", "192.168.1.100/25", "192.168.1.128/25"}
	networksList := []string{"192.168.1.29/32"}
	ctx := context.Background()

	netScanResult := []Result{}
	hostsList := []scannedHost{}
	wp := NewWorkerPool(1, ctx)
	go wp.Run(ctx)
	go wp.StartCollector(ctx, &netScanResult)

	go func() {
		for _, i := range networksList {
			wp.jobs <- NewNetworkScanner(ctx, JobID(i), i)
		}
		close(wp.jobs)
	}()
	<-wp.Done

	close(wp.results)
	<-wp.Done

	for _, i := range netScanResult {
		hostsList = append(hostsList, ToScannedHostList(i.Value)...)
	}

	fmt.Println(len(hostsList))

	portScanResult := []Result{}
	wp = NewWorkerPool(3, ctx)
	go wp.Run(ctx)
	go wp.StartCollector(ctx, &portScanResult)
	for _, h := range hostsList {
		wp.jobs <- NewPortScanner(ctx, JobID(h.Ipv4Addr), h)

	}
	close(wp.jobs)
	<-wp.Done
	close(wp.results)
	<-wp.Done
	fmt.Println(hostsList)
	fmt.Println(portScanResult)

	// for _, i := range portScanResult {
	// 	h := ToScannedHost(i.Value)
	// 	fmt.Println(h.MacAddr, h.Ipv4Addr, h.os, h.ports)

	// }

	// // fmt.Println(portScanResult)
	// fmt.Println(netScanResult)

	// portScanResult := []Result{}
	// wp = NewWorkerPool(3, ctx)
	// go wp.Run(ctx)
	// go wp.StartCollector(ctx, &portScanResult)
	// for _, n := range netScanResult {
	// 	if n.Err != nil {
	// 		fmt.Println("error in network scan result: ", n.Err)
	// 	} else {
	// 		for k, v := range ToMap(n.Value) {
	// 			fmt.Println(k, v)
	// 			wp.jobs <- NewPortScanner(ctx, JobID(v), v)
	// 		}
	// 	}
	// }
	// close(wp.jobs)
	// <-wp.Done
	// close(wp.results)
	// <-wp.Done
	// fmt.Println(portScanResult)

}
