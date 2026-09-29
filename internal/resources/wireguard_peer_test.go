package resources

import (
	"encoding/json"
	"testing"
)

func TestPeerTunnelConfigured(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   bool
	}{
		{"filled", `{"ready":true,"allocated_ip":"10.60.0.2","endpoint_addr":"77.223.120.161:51820","server_public_key":"aFd0="}`, true},
		{"empty object (create race)", `{}`, false},
		{"null", `null`, false},
		{"invalid json", `PENDING`, false},
		{"missing server_public_key", `{"allocated_ip":"10.60.0.2","endpoint_addr":"e:1"}`, false},
		{"missing endpoint_addr", `{"allocated_ip":"10.60.0.2","server_public_key":"k"}`, false},
		{"missing allocated_ip", `{"endpoint_addr":"e:1","server_public_key":"k"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := peerTunnelConfigured(json.RawMessage(tc.status))
			if got != tc.want {
				t.Fatalf("peerTunnelConfigured(%s) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}
