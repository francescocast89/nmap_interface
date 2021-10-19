package main

import (
	"context"
	"fmt"
)

func main() {
	networksList := []string{"192.168.73.0/24"}
	// networksList := []string{"192.168.1.29/32"}
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
