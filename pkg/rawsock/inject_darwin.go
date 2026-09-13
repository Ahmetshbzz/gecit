package rawsock

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/google/gopacket/pcap"
)

// neighborTTL bounds how long a resolved gateway MAC and source address are
// trusted. A router reboot or a SLAAC address change would otherwise leave the
// injector sending to a MAC that no longer answers, failing silently.
const neighborTTL = 60 * time.Second

type ipv6Injector struct {
	iface  string
	handle *pcap.Handle

	mu         sync.Mutex
	srcMAC     net.HardwareAddr
	dstMAC     net.HardwareAddr
	resolvedAt time.Time
}

func newIPv6Injector(iface string) (*ipv6Injector, error) {
	handle, err := pcap.OpenLive(iface, 0, false, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("pcap open %s: %w (run with sudo)", iface, err)
	}
	return &ipv6Injector{iface: iface, handle: handle}, nil
}

func (i *ipv6Injector) close() error {
	i.handle.Close()
	return nil
}

// sourceIPv6 returns the interface's first global (non-link-local, non-loopback)
// IPv6 address together with its hardware address.
func sourceIPv6(iface string) (net.IP, net.HardwareAddr, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, nil, fmt.Errorf("interface %s: %w", iface, err)
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, nil, fmt.Errorf("addresses of %s: %w", iface, err)
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil {
			continue
		}
		ip := ipNet.IP.To16()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		return ip, ifi.HardwareAddr, nil
	}
	return nil, nil, fmt.Errorf("no global IPv6 address on %s", iface)
}

// resolve refreshes the source address and the gateway MAC. The cached values
// are reused until neighborTTL has passed.
func (i *ipv6Injector) resolve() error {
	if i.dstMAC != nil && time.Since(i.resolvedAt) < neighborTTL {
		return nil
	}

	// A global IPv6 address on the interface is a precondition for IPv6
	// injection. The frame's own source address comes from ConnInfo.
	_, srcMAC, err := sourceIPv6(i.iface)
	if err != nil {
		return err
	}

	out, err := exec.Command("netstat", "-rn", "-f", "inet6").Output()
	if err != nil {
		return fmt.Errorf("read IPv6 routing table: %w", err)
	}
	gateway := ipv6DefaultGateway(string(out), i.iface)
	if gateway == "" {
		return fmt.Errorf("no IPv6 default route on %s", i.iface)
	}

	ndpOut, err := exec.Command("ndp", "-an").Output()
	if err != nil {
		return fmt.Errorf("read neighbor table: %w", err)
	}
	dstMAC, err := neighborMAC(string(ndpOut), gateway)
	if err != nil {
		return err
	}

	i.srcMAC = srcMAC
	i.dstMAC = dstMAC
	i.resolvedAt = time.Now()
	return nil
}

func (i *ipv6Injector) send(conn ConnInfo, payload []byte, ttl int) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if err := i.resolve(); err != nil {
		return err
	}

	ipPacket := BuildPacket(conn, payload, ttl)
	frame := BuildEthernetFrame(i.srcMAC, i.dstMAC, ipPacket)
	if err := i.handle.WritePacketData(frame); err != nil {
		return fmt.Errorf("inject IPv6 frame on %s: %w", i.iface, err)
	}
	return nil
}
