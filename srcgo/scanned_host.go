package main

import "fmt"

type ScannedPort struct {
	PortNumber uint16
	Service    string
}

type ScannedOS struct {
	Name     string
	Accuracy int
}

type ScannedHost struct {
	MacAddr  string
	Ipv4Addr string
	os       []ScannedOS
	ports    []ScannedPort
}

const (
	AddrTypeIPV4 = "ipv4"
	AddrTypeMAC  = "mac"
)

func NewScannedHost(h Host) *ScannedHost {
	sh := &ScannedHost{}
	for _, a := range h.Addresses {
		if a.AddrType == AddrTypeMAC {
			sh.MacAddr = a.Addr
		}
		if a.AddrType == AddrTypeIPV4 {
			sh.Ipv4Addr = a.Addr
		}
	}
	if sh.MacAddr == "" {
		sh.MacAddr = fmt.Sprintf("DUMMY_%s", sh.Ipv4Addr)
	}
	return sh
}

func (sh *ScannedHost) AddScannedPort(h Host) {
	for _, p := range h.Ports {
		//fmt.Printf("%0.d : %s \n", p.ID, p.Service.Name)
		sh.ports = append(sh.ports, ScannedPort{p.ID, p.Service.Name})
	}

}

func (sh *ScannedHost) AddScannedOS(h Host) {
	for _, o := range h.OS.Matches {
		//fmt.Printf("%s, %0.d \n", o.Name, o.Accuracy)
		sh.os = append(sh.os, ScannedOS{o.Name, o.Accuracy})
	}

}
