//go:build (darwin || windows) && with_gvisor

package tun

// hopMargin is how many hops short of the measured server distance the fake
// packet dies.
//
// The SYN-ACK measures the return path, and the forward path is neither
// symmetric nor stable: anycast destinations are reached through multipath, so
// the same destination and the same TTL put the fake on the server for some
// flows and not others. Measured against Cloudflare and Google edges, a margin
// of 1 lost every Google flow, 2 lost 6 of 10, and 3 lost none — a flow that
// reaches the server is answered with a RST that kills the connection.
const hopMargin = 3

// fakeTTL returns the hop limit to use for the fake ClientHello.
//
// The fake must travel far enough to reach the DPI but must die before the
// real server: a fake segment that reaches the server is accepted into the
// connection's sequence space, so the server answers the real ClientHello with
// a TLS alert and resets the flow. The server's SYN-ACK carries the hop limit
// left after the path was traversed, which yields the distance to the server
// given the usual initial values of 64, 128 or 255.
//
// observed is 0 when the platform does not report it; configured is then used
// unchanged.
func fakeTTL(configured int, observed uint8) int {
	if observed == 0 {
		return configured
	}

	initial := 64
	switch {
	case observed > 128:
		initial = 255
	case observed > 64:
		initial = 128
	}

	ttl := initial - int(observed) - hopMargin
	if ttl < 1 {
		ttl = 1
	}
	if ttl > configured {
		ttl = configured
	}
	return ttl
}
