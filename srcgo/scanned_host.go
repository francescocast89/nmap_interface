package main

import "fmt"

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
