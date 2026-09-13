//go:build (darwin || windows) && with_gvisor

package tun

import "testing"

func TestFakeTTL(t *testing.T) {
	cases := []struct {
		name       string
		configured int
		observed   uint8
		want       int
	}{
		{"unknown hop limit keeps configured", 8, 0, 8},
		{"server 9 hops away", 8, 55, 6},
		{"server 6 hops away, initial 128", 8, 122, 3},
		{"server 7 hops away, initial 64", 8, 57, 4},
		{"server 10 hops away", 8, 54, 7},
		{"server 14 hops away capped by configured", 8, 50, 8},
		{"never exceeds configured", 3, 55, 3},
		{"clamps to at least 1", 8, 64, 1},
		{"initial 255", 8, 250, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fakeTTL(tc.configured, tc.observed); got != tc.want {
				t.Fatalf("fakeTTL(%d, %d) = %d, want %d", tc.configured, tc.observed, got, tc.want)
			}
		})
	}
}
