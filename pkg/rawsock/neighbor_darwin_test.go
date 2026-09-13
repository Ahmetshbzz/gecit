package rawsock

import (
	"net"
	"testing"
)

const netstatFixture = `Routing tables

Internet6:
Destination                             Gateway                                 Flags               Netif Expire
default                                 fe80::4e2e:feff:fe36:2ce7%en0           UGcg                  en0
default                                 fe80::%utun0                            UGcIg               utun0
default                                 fe80::%utun1                            UGcIg               utun1
::1                                     ::1                                     UHL                   lo0
`

const ndpFixture = `Neighbor                                Linklayer Address  Netif Expire    St Flgs Prbs
2a00:1d35:e2f6:b300:98:275:e642:d37e    26:f8:df:a9:b3:21    en0 permanent R
fe80::1%lo0                             (incomplete)         lo0 permanent R
fe80::1822:56f2:603:ec93%en0            ca:ae:71:a8:a3:10    en0 19h5m42s  S
fe80::4e2e:feff:fe36:2ce7%en0           4c:2e:fe:36:2c:e7    en0 4s        R  R
`

func TestIPv6DefaultGateway_PicksPhysicalInterface(t *testing.T) {
	got := ipv6DefaultGateway(netstatFixture, "en0")
	want := "fe80::4e2e:feff:fe36:2ce7%en0"
	if got != want {
		t.Fatalf("gateway: got %q, want %q", got, want)
	}
}

func TestIPv6DefaultGateway_MissingInterface(t *testing.T) {
	if got := ipv6DefaultGateway(netstatFixture, "en7"); got != "" {
		t.Fatalf("gateway: got %q, want empty", got)
	}
}

func TestNeighborMAC(t *testing.T) {
	got, err := neighborMAC(ndpFixture, "fe80::4e2e:feff:fe36:2ce7%en0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := net.ParseMAC("4c:2e:fe:36:2c:e7")
	if got.String() != want.String() {
		t.Fatalf("MAC: got %v, want %v", got, want)
	}
}

func TestNeighborMAC_IncompleteEntry(t *testing.T) {
	if _, err := neighborMAC(ndpFixture, "fe80::1%lo0"); err == nil {
		t.Fatal("expected error for incomplete neighbor entry")
	}
}

func TestNeighborMAC_NotFound(t *testing.T) {
	if _, err := neighborMAC(ndpFixture, "fe80::dead:beef::1%en0"); err == nil {
		t.Fatal("expected error for unknown neighbor")
	}
}
