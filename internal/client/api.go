package client

import (
	"context"
	"net/url"
)

// Catalog

func (c *Client) ListRegions(ctx context.Context) ([]Region, error) {
	var out []Region
	if err := c.get(ctx, "/api/v1/regions", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetRegion(ctx context.Context, id string) (*Region, error) {
	var out Region
	if err := c.get(ctx, "/api/v1/regions/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListFlavors(ctx context.Context) ([]Flavor, error) {
	var out []Flavor
	if err := c.get(ctx, "/api/v1/flavors", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	var out []Image
	if err := c.get(ctx, "/api/v1/images", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListQuotas(ctx context.Context) ([]QuotaUsage, error) {
	var out []QuotaUsage
	if err := c.get(ctx, "/api/v1/quotas", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListPricing(ctx context.Context, regionID string) ([]RegionPrice, error) {
	path := "/api/v1/pricing"
	if regionID != "" {
		path = withQuery(path, url.Values{"region_id": {regionID}})
	}
	var out []RegionPrice
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Projects

func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	var out Project
	if err := c.post(ctx, "/api/v1/projects", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.get(ctx, "/api/v1/projects", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	var out Project
	if err := c.get(ctx, "/api/v1/projects/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteProject(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/projects/"+id)
}

// SSH keys

func (c *Client) CreateSSHKey(ctx context.Context, req CreateSSHKeyRequest) (*SSHKey, error) {
	var out SSHKey
	if err := c.post(ctx, "/api/v1/ssh-keys", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	var out []SSHKey
	if err := c.get(ctx, "/api/v1/ssh-keys", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetSSHKey(ctx context.Context, id string) (*SSHKey, error) {
	var out SSHKey
	if err := c.get(ctx, "/api/v1/ssh-keys/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/ssh-keys/"+id)
}

// VPCs

func (c *Client) CreateVPC(ctx context.Context, projectID string, req CreateVPCRequest) (*VPC, error) {
	var out VPC
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/vpcs", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetVPC(ctx context.Context, id string) (*VPC, error) {
	var out VPC
	if err := c.get(ctx, "/api/v1/vpcs/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchVPC(ctx context.Context, id string, req PatchVPCRequest) (*VPC, error) {
	var out VPC
	if err := c.patch(ctx, "/api/v1/vpcs/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteVPC(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/vpcs/"+id)
}

// Volumes

func (c *Client) CreateVolume(ctx context.Context, projectID string, req CreateVolumeRequest) (*Volume, error) {
	var out Volume
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/volumes", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetVolume(ctx context.Context, id string) (*Volume, error) {
	var out Volume
	if err := c.get(ctx, "/api/v1/volumes/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchVolume(ctx context.Context, id string, req PatchVolumeRequest) (*Volume, error) {
	var out Volume
	if err := c.patch(ctx, "/api/v1/volumes/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteVolume(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/volumes/"+id)
}

// Network interfaces

func (c *Client) CreateNetworkInterface(ctx context.Context, projectID string, req CreateNetworkInterfaceRequest) (*NetworkInterface, error) {
	var out NetworkInterface
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/network-interfaces", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetNetworkInterface(ctx context.Context, id string) (*NetworkInterface, error) {
	var out NetworkInterface
	if err := c.get(ctx, "/api/v1/network-interfaces/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteNetworkInterface(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/network-interfaces/"+id)
}

// VMs

func (c *Client) CreateVM(ctx context.Context, projectID string, req CreateVMRequest) (*VM, error) {
	var out VM
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/vms", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetVM(ctx context.Context, id string) (*VM, error) {
	var out VM
	if err := c.get(ctx, "/api/v1/vms/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchVM(ctx context.Context, id string, req PatchVMRequest) (*VM, error) {
	var out VM
	if err := c.patch(ctx, "/api/v1/vms/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteVM(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/vms/"+id)
}

// Container images

func (c *Client) CreateContainerImage(ctx context.Context, projectID string, req CreateContainerImageRequest) (*ContainerImage, error) {
	var out ContainerImage
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/container-images", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListContainerImages(ctx context.Context, projectID, regionID string) ([]ContainerImage, error) {
	path := "/api/v1/projects/" + projectID + "/container-images"
	if regionID != "" {
		path = withQuery(path, url.Values{"region_id": {regionID}})
	}
	var out []ContainerImage
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetContainerImage(ctx context.Context, id string) (*ContainerImage, error) {
	var out ContainerImage
	if err := c.get(ctx, "/api/v1/container-images/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteContainerImage(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/container-images/"+id)
}

// Containers

func (c *Client) CreateContainer(ctx context.Context, projectID string, req CreateContainerRequest) (*Container, error) {
	var out Container
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/containers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListContainers(ctx context.Context, projectID, regionID string) ([]Container, error) {
	path := "/api/v1/projects/" + projectID + "/containers"
	if regionID != "" {
		path = withQuery(path, url.Values{"region_id": {regionID}})
	}
	var out []Container
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetContainer(ctx context.Context, id string) (*Container, error) {
	var out Container
	if err := c.get(ctx, "/api/v1/containers/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchContainer(ctx context.Context, id string, req PatchContainerRequest) (*Container, error) {
	var out Container
	if err := c.patch(ctx, "/api/v1/containers/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteContainer(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/containers/"+id)
}

// Gateway routes

func (c *Client) CreateGatewayRoute(ctx context.Context, projectID string, req CreateGatewayRouteRequest) (*GatewayRoute, error) {
	var out GatewayRoute
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/gateway-routes", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListGatewayRoutes(ctx context.Context, projectID, regionID, computeID string) ([]GatewayRoute, error) {
	path := "/api/v1/projects/" + projectID + "/gateway-routes"
	q := url.Values{}
	if regionID != "" {
		q.Set("region_id", regionID)
	}
	if computeID != "" {
		q.Set("compute_id", computeID)
	}
	if len(q) > 0 {
		path = withQuery(path, q)
	}
	var out []GatewayRoute
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetGatewayRoute(ctx context.Context, id string) (*GatewayRoute, error) {
	var out GatewayRoute
	if err := c.get(ctx, "/api/v1/gateway-routes/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchGatewayRoute(ctx context.Context, id string, req PatchGatewayRouteRequest) (*GatewayRoute, error) {
	var out GatewayRoute
	if err := c.patch(ctx, "/api/v1/gateway-routes/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteGatewayRoute(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/gateway-routes/"+id)
}

// WireGuard peers

func (c *Client) CreateWireGuardPeer(ctx context.Context, projectID string, req CreateWireGuardPeerRequest) (*WireGuardPeer, error) {
	var out WireGuardPeer
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/wireguard-peers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetWireGuardPeer(ctx context.Context, id string) (*WireGuardPeer, error) {
	var out WireGuardPeer
	if err := c.get(ctx, "/api/v1/wireguard-peers/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteWireGuardPeer(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/wireguard-peers/"+id)
}

// Load balancers

func (c *Client) CreateLoadBalancer(ctx context.Context, projectID string, req CreateLoadBalancerRequest) (*LoadBalancer, error) {
	var out LoadBalancer
	if err := c.post(ctx, "/api/v1/projects/"+projectID+"/load-balancers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetLoadBalancer(ctx context.Context, id string) (*LoadBalancer, error) {
	var out LoadBalancer
	if err := c.get(ctx, "/api/v1/load-balancers/"+id, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchLoadBalancer(ctx context.Context, id string, req PatchLoadBalancerRequest) (*LoadBalancer, error) {
	var out LoadBalancer
	if err := c.patch(ctx, "/api/v1/load-balancers/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLoadBalancer(ctx context.Context, id string) error {
	return c.delete(ctx, "/api/v1/load-balancers/"+id)
}
