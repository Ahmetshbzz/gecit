package rawsock

import (
	"fmt"
	"net"
	"strings"
)

// ipv6DefaultGateway returns the IPv6 default route's gateway for iface, read
// from `netstat -rn -f inet6` output. The trailing %zone is kept so the caller
// can match it against the neighbor table.
func ipv6DefaultGateway(netstatOut, iface string) string {
	for _, line := range strings.Split(netstatOut, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 || f[0] != "default" || f[3] != iface {
			continue
		}
		return f[1]
	}
	return ""
}

// neighborMAC resolves a link-local gateway address to its MAC using
// `ndp -an` output. Entries without a resolved link-layer address
// ("(incomplete)") are rejected.
func neighborMAC(ndpOut, gateway string) (net.HardwareAddr, error) {
	want := gateway
	if i := strings.Index(want, "%"); i >= 0 {
		want = want[:i]
	}
	for _, line := range strings.Split(ndpOut, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		addr := f[0]
		if i := strings.Index(addr, "%"); i >= 0 {
			addr = addr[:i]
		}
		if !strings.EqualFold(addr, want) {
			continue
		}
		mac, err := net.ParseMAC(f[1])
		if err != nil {
			return nil, fmt.Errorf("parse neighbor MAC %q: %w", f[1], err)
		}
		return mac, nil
	}
	return nil, fmt.Errorf("neighbor %s not found in ndp table", gateway)
}
