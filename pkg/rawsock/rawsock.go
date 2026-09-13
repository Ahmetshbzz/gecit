package rawsock

import (
	"encoding/binary"
	"net"
	"syscall"
)

// ConnInfo holds connection details for crafting fake packets.
type ConnInfo struct {
	SrcIP   net.IP
	DstIP   net.IP
	SrcPort uint16
	DstPort uint16
	Seq     uint32 // TCP sequence number the real data will use
	Ack     uint32 // TCP ACK number (rcv_nxt from the connection)
	// HopLimit is the hop limit observed on the server's SYN-ACK. Zero means
	// unknown (platforms without capture). It bounds the fake packet's TTL:
	// a fake that reaches the server resets the connection.
	HopLimit uint8
}

// RawSocket sends crafted TCP packets with custom TTL.
type RawSocket interface {
	// SendFake sends a fake TCP data packet that DPI will process
	// but the destination server will never receive (low TTL).
	SendFake(conn ConnInfo, payload []byte, ttl int) error
	Close() error
}

// BuildPacket constructs a complete IP+TCP packet with the given payload.
// The IP version follows conn.SrcIP: IPv4 for 4-byte addresses, IPv6 otherwise.
// Used by both Linux and macOS raw socket implementations.
func BuildPacket(conn ConnInfo, payload []byte, ttl int) []byte {
	tcpHdr := buildTCPHeader(conn)
	segment := make([]byte, 0, len(tcpHdr)+len(payload))
	segment = append(segment, tcpHdr...)
	segment = append(segment, payload...)

	var ipHdr, pseudoHdr []byte
	if src4 := conn.SrcIP.To4(); src4 != nil {
		ipHdr = buildIPv4Header(conn, ttl, len(segment))
		pseudoHdr = ipv4PseudoHeader(src4, conn.DstIP.To4(), len(segment))
	} else {
		ipHdr = buildIPv6Header(conn, ttl, len(segment))
		pseudoHdr = ipv6PseudoHeader(conn, len(segment))
	}

	pkt := make([]byte, 0, len(ipHdr)+len(segment))
	pkt = append(pkt, ipHdr...)
	pkt = append(pkt, segment...)

	// Compute TCP checksum (pseudo-header + TCP header + payload).
	cs := Checksum(append(pseudoHdr, segment...))
	tcpChecksumOffset := len(ipHdr) + 16
	pkt[tcpChecksumOffset] = byte(cs >> 8)
	pkt[tcpChecksumOffset+1] = byte(cs)

	return pkt
}

func buildIPv4Header(conn ConnInfo, ttl int, payloadLen int) []byte {
	totalLen := 20 + payloadLen
	hdr := make([]byte, 20)
	hdr[0] = 0x45                                 // Version=4, IHL=5
	ipHeaderPutUint16(hdr[2:4], uint16(totalLen)) // Total length (byte order is platform-dependent)
	ipHeaderPutUint16(hdr[4:6], 0x1234)           // ID
	hdr[8] = byte(ttl)                            // TTL
	hdr[9] = syscall.IPPROTO_TCP                  // Protocol
	copy(hdr[12:16], conn.SrcIP.To4())
	copy(hdr[16:20], conn.DstIP.To4())
	// IP header checksum — required for pcap_sendpacket (kernel won't fill it).
	cs := Checksum(hdr)
	hdr[10] = byte(cs >> 8)
	hdr[11] = byte(cs)
	return hdr
}

// buildIPv6Header builds a 40-byte IPv6 header. IPv6 has no header checksum.
func buildIPv6Header(conn ConnInfo, hopLimit int, payloadLen int) []byte {
	hdr := make([]byte, 40)
	hdr[0] = 0x60                                            // Version=6, Traffic Class=0, Flow Label=0
	binary.BigEndian.PutUint16(hdr[4:6], uint16(payloadLen)) // Payload length
	hdr[6] = syscall.IPPROTO_TCP                             // Next header
	hdr[7] = byte(hopLimit)                                  // Hop limit
	copy(hdr[8:24], conn.SrcIP.To16())
	copy(hdr[24:40], conn.DstIP.To16())
	return hdr
}

func ipv4PseudoHeader(src, dst net.IP, segmentLen int) []byte {
	p := make([]byte, 0, 12)
	p = append(p, src...)
	p = append(p, dst...)
	p = append(p, 0, syscall.IPPROTO_TCP)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(segmentLen))
	return append(p, lenBuf...)
}

// ipv6PseudoHeader builds the 40-byte pseudo-header used for the TCP checksum.
func ipv6PseudoHeader(conn ConnInfo, segmentLen int) []byte {
	p := make([]byte, 0, 40)
	p = append(p, conn.SrcIP.To16()...)
	p = append(p, conn.DstIP.To16()...)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(segmentLen))
	p = append(p, lenBuf...)
	p = append(p, 0, 0, 0, syscall.IPPROTO_TCP)
	return p
}

const (
	etherTypeIPv4  = 0x0800
	etherTypeIPv6  = 0x86dd
	etherHeaderLen = 14
)

// BuildEthernetFrame prepends an Ethernet header to an IP packet for injection
// through pcap. pcap writes at layer 2, so the host routing table does not
// apply — this is what keeps fake packets on the physical interface while the
// IPv6 routes point at the TUN device.
func BuildEthernetFrame(srcMAC, dstMAC net.HardwareAddr, ipPacket []byte) []byte {
	frame := make([]byte, etherHeaderLen+len(ipPacket))
	copy(frame[0:6], dstMAC)
	copy(frame[6:12], srcMAC)
	etherType := uint16(etherTypeIPv4)
	if len(ipPacket) > 0 && ipPacket[0]>>4 == 6 {
		etherType = etherTypeIPv6
	}
	binary.BigEndian.PutUint16(frame[12:14], etherType)
	copy(frame[etherHeaderLen:], ipPacket)
	return frame
}

func buildTCPHeader(conn ConnInfo) []byte {
	hdr := make([]byte, 20)
	binary.BigEndian.PutUint16(hdr[0:2], conn.SrcPort)
	binary.BigEndian.PutUint16(hdr[2:4], conn.DstPort)
	binary.BigEndian.PutUint32(hdr[4:8], conn.Seq)
	binary.BigEndian.PutUint32(hdr[8:12], conn.Ack)
	hdr[12] = 0x50 // Data offset: 5 (20 bytes)
	hdr[13] = 0x18 // Flags: PSH+ACK
	binary.BigEndian.PutUint16(hdr[14:16], 502)
	return hdr
}

// Checksum computes the Internet checksum (RFC 1071).
func Checksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(data); i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 != 0 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}
