package rawsock

import (
	"fmt"
	"syscall"
)

type platformRawSocket struct {
	fd int
	v6 *ipv6Injector
}

func New(iface string) (RawSocket, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, fmt.Errorf("raw socket: %w (run with sudo)", err)
	}

	if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("IP_HDRINCL: %w", err)
	}

	// IPv6 injection goes through pcap: a raw AF_INET6 socket follows the host
	// routing table, which points at the TUN device while gecit is running, so
	// the fake packet would never reach DPI. Opening this here keeps the engine
	// fail-closed at startup rather than silently skipping IPv6 writes.
	v6, err := newIPv6Injector(iface)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return &platformRawSocket{fd: fd, v6: v6}, nil
}

func (s *platformRawSocket) SendFake(conn ConnInfo, payload []byte, ttl int) error {
	if conn.DstIP.To4() == nil {
		return s.v6.send(conn, payload, ttl)
	}

	pkt := BuildPacket(conn, payload, ttl)

	addr := syscall.SockaddrInet4{Port: 0}
	copy(addr.Addr[:], conn.DstIP.To4())

	return syscall.Sendto(s.fd, pkt, 0, &addr)
}

func (s *platformRawSocket) Close() error {
	err := syscall.Close(s.fd)
	if s.v6 != nil {
		if cErr := s.v6.close(); cErr != nil && err == nil {
			err = cErr
		}
	}
	return err
}
