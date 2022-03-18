package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
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
		sh.ports = append(sh.ports, scannedPort{p.ID, p.Service.Name})
	}
}

func (sh *scannedHost) AddScannedOS(h Host) {
	for _, o := range h.OS.Matches {
		sh.os = append(sh.os, scannedOS{o.Name, o.Accuracy})
	}
}

func Subnets(base string, newBits int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(base)
	if err != nil {
		return nil, fmt.Errorf("error parsing CIDR: %s", err)
	}
	if newBits > 32 {
		return nil, fmt.Errorf("number of bits in netmask cannot be greater than 32")
	}

	oldMaskLen, _ := ipnet.Mask.Size()
	if newBits < oldMaskLen {
		fmt.Println(fmt.Errorf("cannot partition /%d address into %d", oldMaskLen, newBits))
		return []string{base}, nil
	}

	oldMaskBin := binary.BigEndian.Uint32(ipnet.Mask)
	newMaskLen, _ := net.CIDRMask(newBits, 32).Size()
	hostsNumber := uint32(math.Pow(2, float64(32-newBits)))

	start := binary.BigEndian.Uint32(ipnet.IP)
	finish := (start & oldMaskBin) | (oldMaskBin ^ 0xffffffff)
	res := make([]string, 0)
	for i := start; i <= finish; i += hostsNumber {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		res = append(res, fmt.Sprintf("%s/%d", ip.String(), newMaskLen))
	}
	return res, nil
}

func main() {
	ctx := context.Background()

	networksList := []string{"192.168.77.111/32", "192.168.77.116/32", "192.168.77.126/32", "192.168.77.118/32", "192.168.77.201/32"}
	//networksList := []string{"192.168.77.0/24"}
	partitionedNetworksList := make([]string, 0)
	for _, i := range networksList {
		res, err := Subnets(i, 26)
		if err != nil {
			fmt.Println(err)
			return
		}
		partitionedNetworksList = append(partitionedNetworksList, res...)
	}
	h := NewNetworkScannerPool(ctx, 4, partitionedNetworksList)
	fmt.Println(h)
	p := NewPortScannerPool(ctx, 4, h)
	fmt.Println(p)

	for {

	}
}
