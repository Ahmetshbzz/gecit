package rawsock

import "testing"

func TestSourceIPv6_LoopbackHasNoGlobalAddress(t *testing.T) {
	if _, _, err := sourceIPv6("lo0"); err == nil {
		t.Fatal("expected error: lo0 has no global IPv6 address")
	}
}

func TestSourceIPv6_PhysicalInterface(t *testing.T) {
	ip, mac, err := sourceIPv6("en0")
	if err != nil {
		t.Skipf("en0 has no global IPv6 address on this machine: %v", err)
	}
	if ip.To4() != nil {
		t.Fatalf("expected an IPv6 address, got %v", ip)
	}
	if len(mac) != 6 {
		t.Fatalf("MAC length: got %d, want 6", len(mac))
	}
}
