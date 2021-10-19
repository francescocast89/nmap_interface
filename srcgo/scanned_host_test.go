package main

import (
	"strings"
	"testing"
)

func TestNewScannedHostWithBothIPV4AndMAC(t *testing.T) {
	wantIPV4 := "1.2.3.4.5"
	wantMAC := "11:22:33:44:55:66"
	h := Host{
		Addresses: []Address{
			{
				Addr:     wantIPV4,
				AddrType: AddrTypeIPV4,
			},
			{
				Addr:     wantMAC,
				AddrType: AddrTypeMAC,
			},
		},
	}
	scannedHost := NewScannedHost(h)
	if scannedHost.Ipv4Addr != wantIPV4 {
		t.Errorf("wrong Ipv4Addr: got %s, want %s", scannedHost.Ipv4Addr, wantIPV4)
	}
	if scannedHost.MacAddr != wantMAC {
		t.Errorf("wrong MacAddr: got %s, want %s", scannedHost.MacAddr, wantMAC)
	}
}

func TestNewScannedHostWithoutMAC(t *testing.T) {
	wantIPV4 := "1.2.3.4.5"
	h := Host{
		Addresses: []Address{
			{
				Addr:     wantIPV4,
				AddrType: AddrTypeIPV4,
			},
		},
	}
	scannedHost := NewScannedHost(h)
	if scannedHost.Ipv4Addr != wantIPV4 {
		t.Errorf("wrong Ipv4Addr: got %s, want %s", scannedHost.Ipv4Addr, wantIPV4)
	}
	if !strings.Contains(scannedHost.MacAddr, "DUMMY") {
		t.Errorf("wrong MacAddr: should contain 'DUMMY', got %s", scannedHost.MacAddr)
	}
}
