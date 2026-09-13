package capture

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// IPv6 must not be rejected by the kernel filter: an expression that combined
// the bare `tcp` primitive with the `tcp[tcpflags]` index compiled to an
// IPv4-only branch, dropping every IPv6 SYN-ACK before Go ever saw it.
func TestSynAckFilter_AcceptsIPv6(t *testing.T) {
	insns, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 65536, synAckFilter(443))
	if err != nil {
		t.Fatalf("compile filter: %v", err)
	}

	found := false
	for idx, in := range insns {
		if in.Code != 0x15 || in.K != 0x86dd {
			continue
		}
		next := idx + 1 + int(in.Jt)
		if next >= len(insns) {
			t.Fatalf("IPv6 branch jumps past the end of the program")
		}
		if insns[next].Code == 0x06 && insns[next].K == 0 {
			t.Fatalf("IPv6 branch rejects the packet at instruction %d", next)
		}
		found = true
	}
	if !found {
		t.Fatal("filter has no IPv6 ethertype branch")
	}
}

func TestSynAckFilter_AcceptsIPv4(t *testing.T) {
	insns, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, 65536, synAckFilter(443))
	if err != nil {
		t.Fatalf("compile filter: %v", err)
	}
	found := false
	for _, in := range insns {
		if in.Code == 0x15 && in.K == 0x0800 {
			found = true
		}
	}
	if !found {
		t.Fatal("filter has no IPv4 ethertype branch")
	}
}

func TestProcessPacket_IPv6SYNACK(t *testing.T) {
	c := &pcapCapture{ports: map[uint16]bool{443: true}}

	var got *ConnectionEvent
	c.processPacket(ipv6SYNACKPacket(t), func(evt ConnectionEvent) { got = &evt })

	if got == nil {
		t.Fatal("no event emitted for IPv6 SYN-ACK")
	}
	if !got.SrcIP.Equal(net.ParseIP("2a00:1d35:e2f6:b300:2a00:2a00:2a00:2a00")) {
		t.Fatalf("client IP: got %v", got.SrcIP)
	}
	if !got.DstIP.Equal(net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1")) {
		t.Fatalf("server IP: got %v", got.DstIP)
	}
	if got.SrcPort != 51234 {
		t.Fatalf("client port: got %d, want 51234", got.SrcPort)
	}
	if got.DstPort != 443 {
		t.Fatalf("server port: got %d, want 443", got.DstPort)
	}
	// evt.Seq = tcp.Ack = 77 (our snd_nxt); evt.Ack = tcp.Seq + 1 = 76 + 1 = 77.
	if got.Seq != 77 || got.Ack != 77 {
		t.Fatalf("seq/ack: got %d/%d, want 77/77", got.Seq, got.Ack)
	}
	// The hop limit bounds the fake packet's TTL, so it must survive the trip.
	if got.HopLimit != 64 {
		t.Fatalf("hop limit: got %d, want 64", got.HopLimit)
	}
}

func TestProcessPacket_IPv4SYNACK_RecordsTTL(t *testing.T) {
	c := &pcapCapture{ports: map[uint16]bool{443: true}}

	var got *ConnectionEvent
	c.processPacket(ipv4SYNACKPacket(t), func(evt ConnectionEvent) { got = &evt })

	if got == nil {
		t.Fatal("no event emitted for IPv4 SYN-ACK")
	}
	if got.HopLimit != 53 {
		t.Fatalf("TTL: got %d, want 53", got.HopLimit)
	}
}

func ipv4SYNACKPacket(t *testing.T) gopacket.Packet {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip4 := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      53,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    net.IPv4(108, 138, 192, 108),
		DstIP:    net.IPv4(192, 168, 1, 103),
	}
	tcp := &layers.TCP{
		SrcPort: 443,
		DstPort: 51234,
		Seq:     76,
		Ack:     77,
		SYN:     true,
		ACK:     true,
		Window:  65535,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip4); err != nil {
		t.Fatalf("set network layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip4, tcp); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

// ipv6SYNACKPacket builds a server->client SYN-ACK: server port 443, seq 76,
// ack 77, so the client's snd_nxt is 77 and rcv_nxt is 77.
func ipv6SYNACKPacket(t *testing.T) gopacket.Packet {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv6,
	}
	ip6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolTCP,
		HopLimit:   64,
		SrcIP:      net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1"),
		DstIP:      net.ParseIP("2a00:1d35:e2f6:b300:2a00:2a00:2a00:2a00"),
	}
	tcp := &layers.TCP{
		SrcPort: 443,
		DstPort: 51234,
		Seq:     76,
		Ack:     77,
		SYN:     true,
		ACK:     true,
		Window:  65535,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip6); err != nil {
		t.Fatalf("set network layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip6, tcp); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}
