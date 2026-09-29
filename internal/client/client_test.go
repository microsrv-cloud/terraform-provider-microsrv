package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
)

func TestListRegions(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/regions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("auth %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode([]client.Region{{ID: "r1", Code: "ru-yndx-central", Name: "YC", Enabled: true}})
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.ListRegions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Code != "ru-yndx-central" {
		t.Fatalf("%+v", out)
	}
}

func TestCreateVPCAndAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.VPC{ID: "v1", State: "PENDING", CIDR: "10.0.0.0/24"})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"quota exceeded"}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	vpc, err := c.CreateVPC(context.Background(), "p1", client.CreateVPCRequest{RegionID: "r1", CIDR: "10.0.0.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if vpc.ID != "v1" {
		t.Fatalf("%+v", vpc)
	}
	_, err = c.GetVPC(context.Background(), "v1")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok || !apiErr.IsForbidden() || apiErr.Message != "quota exceeded" {
		t.Fatalf("%v", err)
	}
}

func TestUnauthorizedCarriesMintingHint(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "stale-jwt"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ListRegions(context.Background())
	apiErr, ok := err.(*client.APIError)
	if !ok || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(apiErr.Message, "invalid token") || !strings.Contains(apiErr.Message, "sel_") {
		t.Fatalf("message lacks minting hint: %q", apiErr.Message)
	}
}

func TestCreateContainerImageAndContainer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/projects/p1/container-images":
			if r.Method != http.MethodPost {
				t.Fatalf("method %s", r.Method)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.ContainerImage{ID: "ci1", ProjectID: "p1", Ref: "nginx:1.27", State: "PENDING"})
		case "/api/v1/projects/p1/containers":
			if r.Method != http.MethodPost {
				t.Fatalf("method %s", r.Method)
			}
			var creq client.CreateContainerRequest
			if err := json.NewDecoder(r.Body).Decode(&creq); err != nil {
				t.Fatal(err)
			}
			if len(creq.SSHKeyIDs) != 1 || creq.SSHKeyIDs[0] != "sk1" {
				t.Fatalf("%+v", creq)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.Container{ID: "c1", ProjectID: "p1", Name: "web", Command: []string{"/bin/sh"}, SSHKeyIDs: creq.SSHKeyIDs, State: "PENDING"})
		case "/api/v1/containers/c1":
			if r.Method != http.MethodPatch {
				t.Fatalf("method %s", r.Method)
			}
			var preq client.PatchContainerRequest
			if err := json.NewDecoder(r.Body).Decode(&preq); err != nil {
				t.Fatal(err)
			}
			if preq.SSHKeyIDs == nil || len(*preq.SSHKeyIDs) != 1 || (*preq.SSHKeyIDs)[0] != "sk1" {
				t.Fatalf("ssh_key_ids %+v", preq.SSHKeyIDs)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.Container{ID: "c1", ProjectID: "p1", Name: "web", State: "UPDATING"})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	img, err := c.CreateContainerImage(context.Background(), "p1", client.CreateContainerImageRequest{RegionID: "r1", Ref: "nginx:1.27"})
	if err != nil {
		t.Fatal(err)
	}
	if img.ID != "ci1" {
		t.Fatalf("%+v", img)
	}

	ctr, err := c.CreateContainer(context.Background(), "p1", client.CreateContainerRequest{
		RegionID:  "r1",
		FlavorID:  "f1",
		ImageID:   "ci1",
		Name:      "web",
		Command:   []string{"/bin/sh"},
		VolumeIDs: []string{"v1"},
		SSHKeyIDs: []string{"sk1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ctr.ID != "c1" || len(ctr.Command) != 1 || len(ctr.SSHKeyIDs) != 1 {
		t.Fatalf("%+v", ctr)
	}

	name := "web"
	fid := "f2"
	sshKeys := []string{"sk1"}
	upd, err := c.PatchContainer(context.Background(), "c1", client.PatchContainerRequest{Name: &name, FlavorID: &fid, SSHKeyIDs: &sshKeys})
	if err != nil {
		t.Fatal(err)
	}
	if upd.ID != "c1" {
		t.Fatalf("%+v", upd)
	}
}

// The VM create must carry the boot disk as boot_volume_id and never as
// clone_from/root_size_mib, and must not send the frontend-only gateway
// convenience fields (expose_ssh/expose_http/target_port) at all.
func TestCreateVMBootVolumeRouting(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/api/v1/projects/p1/vms":
			for _, forbidden := range []string{"clone_from", "root_size_mib", "expose_ssh", "expose_http", "target_port"} {
				if _, ok := body[forbidden]; ok {
					t.Errorf("vm create must not carry %s: %s", forbidden, body[forbidden])
				}
			}
			var boot string
			if err := json.Unmarshal(body["boot_volume_id"], &boot); err != nil || boot != "vol1" {
				t.Errorf("boot_volume_id %q (err %v)", boot, err)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.VM{ID: "vm1", State: "PENDING"})
		case "/api/v1/projects/p1/volumes":
			// Boot volumes are explicit resources: clone_from implies bootable.
			var clone string
			if err := json.Unmarshal(body["clone_from"], &clone); err != nil || clone != "ubuntu-2404" {
				t.Errorf("clone_from %q (err %v)", clone, err)
			}
			var bootable bool
			if err := json.Unmarshal(body["bootable"], &bootable); err != nil || !bootable {
				t.Errorf("bootable %v (err %v)", bootable, err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(client.Volume{ID: "vol1", State: "PENDING", CloneFrom: clone, Bootable: true})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	vol, err := c.CreateVolume(context.Background(), "p1", client.CreateVolumeRequest{RegionID: "r1", SizeMiB: 20480, CloneFrom: "ubuntu-2404", Bootable: true})
	if err != nil || vol.ID != "vol1" {
		t.Fatalf("volume %+v err %v", vol, err)
	}
	vm, err := c.CreateVM(context.Background(), "p1", client.CreateVMRequest{
		RegionID:            "r1",
		FlavorID:            "f1",
		Class:               "shared",
		Name:                "web",
		NetworkInterfaceIDs: []string{"ni1"},
		SSHKeyIDs:           []string{"sk1"},
		BootVolumeID:        vol.ID,
	})
	if err != nil || vm.ID != "vm1" {
		t.Fatalf("vm %+v err %v", vm, err)
	}
}

func TestListContainerImagesWithRegionFilter(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/projects/p1/container-images" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("region_id"); got != "r1" {
			t.Fatalf("region_id %q", got)
		}
		_ = json.NewEncoder(w).Encode([]client.ContainerImage{{ID: "ci1", Ref: "nginx:1.27", State: "READY"}})
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.ListContainerImages(context.Background(), "p1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Ref != "nginx:1.27" {
		t.Fatalf("%+v", out)
	}
}

func TestLoadBalancerLifecycle(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/p1/load-balancers":
			var req client.CreateLoadBalancerRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.VPCID != "vpc1" || req.VIP != "10.44.0.10" || len(req.Listeners) != 1 || len(req.Backends) != 1 {
				t.Fatalf("%+v", req)
			}
			if req.Listeners[0].Protocol != "tcp" || req.Listeners[0].ListenPort != 80 || req.Listeners[0].TargetPort != 8080 {
				t.Fatalf("%+v", req)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.LoadBalancer{ID: "lb1", ProjectID: "p1", RegionID: "r1", VPCID: req.VPCID, VIP: req.VIP, Listeners: req.Listeners, Backends: req.Backends, HealthCheck: client.LBHealthCheck{Protocol: "tcp"}, State: "PENDING"})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/load-balancers/lb1":
			var preq client.PatchLoadBalancerRequest
			if err := json.NewDecoder(r.Body).Decode(&preq); err != nil {
				t.Fatal(err)
			}
			if preq.Backends == nil || len(*preq.Backends) != 1 || (*preq.Backends)[0] != "c1" {
				t.Fatalf("%+v", preq)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.LoadBalancer{ID: "lb1", ProjectID: "p1", RegionID: "r1", VPCID: "vpc1", VIP: "10.44.0.10", Listeners: []client.LBListener{{Protocol: "tcp", ListenPort: 80, TargetPort: 8080, Algorithm: "round_robin"}}, Backends: *preq.Backends, HealthCheck: client.LBHealthCheck{Protocol: "tcp"}, State: "UPDATING"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/load-balancers/lb1":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(client.LoadBalancer{ID: "lb1", ProjectID: "p1", RegionID: "r1", VPCID: "vpc1", VIP: "10.44.0.10", Listeners: []client.LBListener{{Protocol: "tcp", ListenPort: 80, TargetPort: 8080, Algorithm: "round_robin"}}, Backends: []string{"vm1"}, HealthCheck: client.LBHealthCheck{Protocol: "tcp"}, State: "READY", Status: json.RawMessage(`{"ready":true}`)})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/load-balancers/lb1":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.LoadBalancer{ID: "lb1", State: "DELETING"})
		default:
			t.Fatalf("unhandled %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	lb, err := c.CreateLoadBalancer(context.Background(), "p1", client.CreateLoadBalancerRequest{
		VPCID:     "vpc1",
		VIP:       "10.44.0.10",
		Listeners: []client.LBListener{{Protocol: "tcp", ListenPort: 80, TargetPort: 8080}},
		Backends:  []string{"vm1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if lb.ID != "lb1" || lb.VIP != "10.44.0.10" {
		t.Fatalf("%+v", lb)
	}

	got, err := c.GetLoadBalancer(context.Background(), "lb1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "READY" || len(got.Listeners) != 1 || got.Listeners[0].Algorithm != "round_robin" {
		t.Fatalf("%+v", got)
	}

	backends := []string{"c1"}
	if _, err := c.PatchLoadBalancer(context.Background(), "lb1", client.PatchLoadBalancerRequest{Backends: &backends}); err != nil {
		t.Fatal(err)
	}

	if err := c.DeleteLoadBalancer(context.Background(), "lb1"); err != nil {
		t.Fatal(err)
	}
}

func TestGatewayRouteLifecycle(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/p1/gateway-routes":
			var req client.CreateGatewayRouteRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.ComputeID != "vm1" || req.Protocol != "ssh" {
				t.Fatalf("%+v", req)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.GatewayRoute{ID: "gr1", ProjectID: "p1", RegionID: "r1", ComputeID: req.ComputeID, Protocol: req.Protocol, Enabled: true, State: "PENDING"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/gateway-routes/gr1":
			_ = json.NewEncoder(w).Encode(client.GatewayRoute{ID: "gr1", ProjectID: "p1", RegionID: "r1", ComputeID: "vm1", Protocol: "ssh", Enabled: true, State: "READY", Status: json.RawMessage(`{"ready":true}`)})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/gateway-routes/gr1":
			var req client.PatchGatewayRouteRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Comment == nil || *req.Comment != "hello" {
				t.Fatalf("%+v", req)
			}
			if req.TargetPort == nil || *req.TargetPort != 0 {
				t.Fatalf("target_port %+v", req.TargetPort)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.GatewayRoute{ID: "gr1", ProjectID: "p1", RegionID: "r1", ComputeID: "vm1", Protocol: "ssh", Comment: "hello", Enabled: true, State: "UPDATING"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/gateway-routes/gr1":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.GatewayRoute{ID: "gr1", State: "DELETING"})
		default:
			t.Fatalf("unhandled %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	route, err := c.CreateGatewayRoute(context.Background(), "p1", client.CreateGatewayRouteRequest{
		ComputeID: "vm1",
		Protocol:  "ssh",
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.ID != "gr1" || !route.Enabled {
		t.Fatalf("%+v", route)
	}

	got, err := c.GetGatewayRoute(context.Background(), "gr1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "READY" {
		t.Fatalf("%+v", got)
	}

	comment := "hello"
	targetPort := 0 // clear back to the console default
	if _, err := c.PatchGatewayRoute(context.Background(), "gr1", client.PatchGatewayRouteRequest{Comment: &comment, TargetPort: &targetPort}); err != nil {
		t.Fatal(err)
	}

	if err := c.DeleteGatewayRoute(context.Background(), "gr1"); err != nil {
		t.Fatal(err)
	}
}

func TestGatewayRouteSNIPassthrough(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/p1/gateway-routes" {
			var req client.CreateGatewayRouteRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.Protocol != "tls" || req.SNIDomain != "app.example.com" {
				t.Fatalf("%+v", req)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.GatewayRoute{ID: "gr1", ProjectID: "p1", RegionID: "r1", ComputeID: req.ComputeID, Protocol: req.Protocol, SNIDomain: req.SNIDomain, Enabled: true, State: "READY"})
			return
		}
		t.Fatalf("unhandled %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	route, err := c.CreateGatewayRoute(context.Background(), "p1", client.CreateGatewayRouteRequest{
		ComputeID: "vm1",
		Protocol:  "tls",
		SNIDomain: "app.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.SNIDomain != "app.example.com" {
		t.Fatalf("%+v", route)
	}
}

func TestWireGuardPeerLifecycle(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/p1/wireguard-peers":
			var req client.CreateWireGuardPeerRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.NetworkInterfaceID != "ni1" || req.PublicKey != "pub" {
				t.Fatalf("%+v", req)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.WireGuardPeer{ID: "wg1", ProjectID: "p1", RegionID: "r1", NetworkInterfaceID: req.NetworkInterfaceID, PublicKey: req.PublicKey, PersistentKeepalive: 25, Enabled: true, State: "PENDING"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wireguard-peers/wg1":
			_ = json.NewEncoder(w).Encode(client.WireGuardPeer{ID: "wg1", ProjectID: "p1", RegionID: "r1", NetworkInterfaceID: "ni1", PublicKey: "pub", PersistentKeepalive: 25, Enabled: true, State: "READY", Status: json.RawMessage(`{"ready":true,"allocated_ip":"10.50.0.100"}`)})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/wireguard-peers/wg1":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.WireGuardPeer{ID: "wg1", State: "DELETING"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/wireguard-peers/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		default:
			t.Fatalf("unhandled %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	en := true
	peer, err := c.CreateWireGuardPeer(context.Background(), "p1", client.CreateWireGuardPeerRequest{
		NetworkInterfaceID: "ni1",
		PublicKey:          "pub",
		Enabled:            &en,
	})
	if err != nil {
		t.Fatal(err)
	}
	if peer.ID != "wg1" || !peer.Enabled || peer.PersistentKeepalive != 25 {
		t.Fatalf("%+v", peer)
	}

	got, err := c.GetWireGuardPeer(context.Background(), "wg1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "READY" {
		t.Fatalf("%+v", got)
	}

	if _, err := c.GetWireGuardPeer(context.Background(), "missing"); err == nil {
		t.Fatal("expected 404")
	} else if apiErr, ok := err.(*client.APIError); !ok || !apiErr.IsNotFound() {
		t.Fatalf("%v", err)
	}

	if err := c.DeleteWireGuardPeer(context.Background(), "wg1"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateNetworkInterfacePinsIP(t *testing.T) {
	t.Parallel()
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = nil
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(client.NetworkInterface{ID: "ni1", IPAddress: "10.0.0.5", State: "PENDING"})
	}))
	defer srv.Close()

	c, err := client.New(client.Config{Endpoint: srv.URL, Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	ni, err := c.CreateNetworkInterface(context.Background(), "p1", client.CreateNetworkInterfaceRequest{VPCID: "v1", IPAddress: "10.0.0.5"})
	if err != nil {
		t.Fatal(err)
	}
	if ni.IPAddress != "10.0.0.5" {
		t.Fatalf("response %+v", ni)
	}
	if got["ip_address"] != "10.0.0.5" {
		t.Fatalf("pinned ip not sent: %#v", got)
	}

	// Omitting the pin must not serialize ip_address, so the API auto-allocates.
	if _, err := c.CreateNetworkInterface(context.Background(), "p1", client.CreateNetworkInterfaceRequest{VPCID: "v1"}); err != nil {
		t.Fatal(err)
	}
	if _, present := got["ip_address"]; present {
		t.Fatalf("empty ip_address serialized: %#v", got)
	}
}
