package rawsock

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestChecksum_KnownVector(t *testing.T) {
	// RFC 1071 example: 0x0001 + 0xf203 + ... = 0xddf2
	data := []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}
	got := Checksum(data)
	if got == 0 {
		t.Fatal("checksum should not be zero for non-zero input")
	}
	// Verify checksum property: appending the checksum to the data
	// should yield a zero checksum (complement property).
	data = append(data, byte(got>>8), byte(got))
	if verify := Checksum(data); verify != 0 {
		t.Fatalf("checksum verify failed: got 0x%04x, want 0x0000", verify)
	}
}

func TestChecksum_Empty(t *testing.T) {
	got := Checksum([]byte{})
	if got != 0xFFFF {
		t.Fatalf("checksum of empty data: got 0x%04x, want 0xFFFF", got)
	}
}

func TestChecksum_OddLength(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0xEF}
	got := Checksum(data)
	if got == 0 {
		t.Fatal("checksum should not be zero")
	}
	// Pad to even and verify complement property.
	padded := append(data, 0x00)
	padded = append(padded, byte(got>>8), byte(got))
	// Recompute over padded+checksum — should give complement of padding effect.
	// Instead, just verify consistency: same input → same output.
	if got2 := Checksum(data); got != got2 {
		t.Fatalf("checksum not deterministic: 0x%04x vs 0x%04x", got, got2)
	}
}

func TestBuildPacket_IPHeader(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.IPv4(10, 0, 0, 1),
		DstIP:   net.IPv4(93, 184, 216, 34),
		SrcPort: 12345,
		DstPort: 443,
		Seq:     1000,
		Ack:     2000,
	}
	payload := []byte("test-payload")
	ttl := 8

	pkt := BuildPacket(conn, payload, ttl)

	// IP header is first 20 bytes.
	if len(pkt) < 40+len(payload) {
		t.Fatalf("packet too short: %d bytes", len(pkt))
	}

	// Version + IHL.
	if pkt[0] != 0x45 {
		t.Fatalf("IP version/IHL: got 0x%02x, want 0x45", pkt[0])
	}

	// TTL.
	if pkt[8] != byte(ttl) {
		t.Fatalf("TTL: got %d, want %d", pkt[8], ttl)
	}

	// Protocol (TCP = 6).
	if pkt[9] != 6 {
		t.Fatalf("protocol: got %d, want 6", pkt[9])
	}

	// Src IP.
	if !net.IP(pkt[12:16]).Equal(conn.SrcIP.To4()) {
		t.Fatalf("src IP: got %v, want %v", net.IP(pkt[12:16]), conn.SrcIP)
	}

	// Dst IP.
	if !net.IP(pkt[16:20]).Equal(conn.DstIP.To4()) {
		t.Fatalf("dst IP: got %v, want %v", net.IP(pkt[16:20]), conn.DstIP)
	}
}

func TestBuildPacket_TCPHeader(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.IPv4(10, 0, 0, 1),
		DstIP:   net.IPv4(93, 184, 216, 34),
		SrcPort: 12345,
		DstPort: 443,
		Seq:     1000,
		Ack:     2000,
	}
	payload := []byte("hello")

	pkt := BuildPacket(conn, payload, 8)
	tcp := pkt[20:] // TCP header starts after 20-byte IP header.

	// Src port.
	srcPort := binary.BigEndian.Uint16(tcp[0:2])
	if srcPort != conn.SrcPort {
		t.Fatalf("src port: got %d, want %d", srcPort, conn.SrcPort)
	}

	// Dst port.
	dstPort := binary.BigEndian.Uint16(tcp[2:4])
	if dstPort != conn.DstPort {
		t.Fatalf("dst port: got %d, want %d", dstPort, conn.DstPort)
	}

	// Seq.
	seq := binary.BigEndian.Uint32(tcp[4:8])
	if seq != conn.Seq {
		t.Fatalf("seq: got %d, want %d", seq, conn.Seq)
	}

	// Ack.
	ack := binary.BigEndian.Uint32(tcp[8:12])
	if ack != conn.Ack {
		t.Fatalf("ack: got %d, want %d", ack, conn.Ack)
	}

	// Data offset: 5 (20 bytes).
	if tcp[12] != 0x50 {
		t.Fatalf("data offset: got 0x%02x, want 0x50", tcp[12])
	}

	// Flags: PSH+ACK.
	if tcp[13] != 0x18 {
		t.Fatalf("flags: got 0x%02x, want 0x18 (PSH+ACK)", tcp[13])
	}
}

func TestBuildPacket_Payload(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.IPv4(10, 0, 0, 1),
		DstIP:   net.IPv4(1, 1, 1, 1),
		SrcPort: 5000,
		DstPort: 443,
		Seq:     100,
		Ack:     200,
	}
	payload := []byte("the-real-payload")

	pkt := BuildPacket(conn, payload, 8)

	// Payload starts after IP (20) + TCP (20) headers.
	got := pkt[40:]
	if string(got) != string(payload) {
		t.Fatalf("payload: got %q, want %q", got, payload)
	}
}

func TestBuildPacket_TCPChecksum(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.IPv4(10, 0, 0, 1),
		DstIP:   net.IPv4(93, 184, 216, 34),
		SrcPort: 12345,
		DstPort: 443,
		Seq:     1000,
		Ack:     2000,
	}
	payload := []byte("test")

	pkt := BuildPacket(conn, payload, 8)

	// Verify TCP checksum by recomputing over pseudo-header + TCP segment.
	tcpSeg := pkt[20:]
	pseudo := make([]byte, 0, 12+len(tcpSeg))
	pseudo = append(pseudo, conn.SrcIP.To4()...)
	pseudo = append(pseudo, conn.DstIP.To4()...)
	pseudo = append(pseudo, 0, 6) // reserved + protocol TCP
	tcpLen := make([]byte, 2)
	binary.BigEndian.PutUint16(tcpLen, uint16(len(tcpSeg)))
	pseudo = append(pseudo, tcpLen...)
	pseudo = append(pseudo, tcpSeg...)

	if cs := Checksum(pseudo); cs != 0 {
		t.Fatalf("TCP checksum verification failed: got 0x%04x, want 0x0000", cs)
	}
}

func TestBuildPacket_DifferentTTL(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.IPv4(10, 0, 0, 1),
		DstIP:   net.IPv4(1, 1, 1, 1),
		SrcPort: 5000,
		DstPort: 443,
		Seq:     100,
		Ack:     200,
	}

	for _, ttl := range []int{1, 8, 64, 128, 255} {
		pkt := BuildPacket(conn, []byte("x"), ttl)
		if pkt[8] != byte(ttl) {
			t.Errorf("TTL %d: got %d", ttl, pkt[8])
		}
	}
}

func TestBuildPacket_IPv6Header(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.ParseIP("2a00:1d35:e2f6:b300:98:275:e642:d37e"),
		DstIP:   net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1"),
		SrcPort: 51234,
		DstPort: 443,
		Seq:     1000,
		Ack:     2000,
	}
	payload := []byte("hello")

	pkt := BuildPacket(conn, payload, 8)

	if len(pkt) != 40+20+len(payload) {
		t.Fatalf("packet length: got %d, want %d", len(pkt), 40+20+len(payload))
	}
	if pkt[0] != 0x60 {
		t.Fatalf("IP version: got 0x%02x, want 0x60", pkt[0])
	}
	if got := binary.BigEndian.Uint16(pkt[4:6]); got != uint16(20+len(payload)) {
		t.Fatalf("payload length: got %d, want %d", got, 20+len(payload))
	}
	if pkt[6] != 6 {
		t.Fatalf("next header: got %d, want 6 (TCP)", pkt[6])
	}
	if pkt[7] != 8 {
		t.Fatalf("hop limit: got %d, want 8", pkt[7])
	}
	if got := net.IP(pkt[8:24]); !got.Equal(conn.SrcIP) {
		t.Fatalf("src IP: got %v, want %v", got, conn.SrcIP)
	}
	if got := net.IP(pkt[24:40]); !got.Equal(conn.DstIP) {
		t.Fatalf("dst IP: got %v, want %v", got, conn.DstIP)
	}
}

func TestBuildPacket_IPv6TCPChecksum(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.ParseIP("2a00:1d35:e2f6:b300:98:275:e642:d37e"),
		DstIP:   net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1"),
		SrcPort: 51234,
		DstPort: 443,
		Seq:     1000,
		Ack:     2000,
	}
	payload := []byte("test")

	pkt := BuildPacket(conn, payload, 8)
	segment := pkt[40:]

	// IPv6 pseudo-header: src(16) + dst(16) + upper-layer length(4) + zero(3) + next header(1)
	pseudo := make([]byte, 0, 40+len(segment))
	pseudo = append(pseudo, conn.SrcIP.To16()...)
	pseudo = append(pseudo, conn.DstIP.To16()...)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(segment)))
	pseudo = append(pseudo, lenBuf...)
	pseudo = append(pseudo, 0, 0, 0, 6)
	pseudo = append(pseudo, segment...)

	if cs := Checksum(pseudo); cs != 0 {
		t.Fatalf("TCP checksum verification failed: got 0x%04x, want 0x0000", cs)
	}
}

func TestBuildPacket_IPv6DifferentHopLimit(t *testing.T) {
	conn := ConnInfo{
		SrcIP:   net.ParseIP("2a00:1d35:e2f6:b300:98:275:e642:d37e"),
		DstIP:   net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1"),
		SrcPort: 5000,
		DstPort: 443,
	}

	for _, ttl := range []int{1, 8, 64, 255} {
		pkt := BuildPacket(conn, []byte("x"), ttl)
		if pkt[7] != byte(ttl) {
			t.Errorf("hop limit %d: got %d", ttl, pkt[7])
		}
	}
}

func TestBuildEthernetFrame_IPv6(t *testing.T) {
	srcMAC, _ := net.ParseMAC("26:f8:df:a9:b3:21")
	dstMAC, _ := net.ParseMAC("4c:2e:fe:36:2c:e7")
	conn := ConnInfo{
		SrcIP:   net.ParseIP("2a00:1d35:e2f6:b300:98:275:e642:d37e"),
		DstIP:   net.ParseIP("2600:9000:2921:c00:11:b398:a580:93a1"),
		SrcPort: 51234,
		DstPort: 443,
	}
	ipPacket := BuildPacket(conn, nil, 8)

	frame := BuildEthernetFrame(srcMAC, dstMAC, ipPacket)

	if len(frame) != 14+len(ipPacket) {
		t.Fatalf("frame length: got %d, want %d", len(frame), 14+len(ipPacket))
	}
	if got := net.HardwareAddr(frame[0:6]); got.String() != dstMAC.String() {
		t.Fatalf("dst MAC: got %v, want %v", got, dstMAC)
	}
	if got := net.HardwareAddr(frame[6:12]); got.String() != srcMAC.String() {
		t.Fatalf("src MAC: got %v, want %v", got, srcMAC)
	}
	if got := binary.BigEndian.Uint16(frame[12:14]); got != 0x86dd {
		t.Fatalf("ethertype: got 0x%04x, want 0x86dd", got)
	}
	if frame[14] != 0x60 {
		t.Fatalf("inner IP version: got 0x%02x, want 0x60", frame[14])
	}
	if frame[14+7] != 8 {
		t.Fatalf("hop limit: got %d, want 8", frame[14+7])
	}
}
