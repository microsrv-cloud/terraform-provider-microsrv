package client

import "encoding/json"

// Lifecycle states.
const (
	StatePending      = "PENDING"
	StateProvisioning = "PROVISIONING"
	StateReady        = "READY"
	StateCheckpointed = "CHECKPOINTED"
	StateUpdating     = "UPDATING"
	StateDeleting     = "DELETING"
	StateFailed       = "FAILED"
)

// IsReadyState reports whether state is operationally ready (READY or
// CHECKPOINTED). Checkpointed workloads are paused on disk but still
// provisioned; they satisfy existence gates (gateway routes, etc.).
func IsReadyState(state string) bool {
	return state == StateReady || state == StateCheckpointed
}

type Region struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	OwnerUserID string `json:"owner_user_id"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SSHKey struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type QuotaUsage struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	ResourceKind string `json:"resource_kind"`
	HardLimit    int64  `json:"hard_limit"`
	Usage        int64  `json:"usage"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Flavor struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	VCPUs     int    `json:"vcpus"`
	MemoryMiB int64  `json:"memory_mib"`
	CreatedAt string `json:"created_at"`
	// Available and Reason are enriched from a live provisioning check:
	// whether this flavor can currently be admitted by the cluster
	// scheduler. Available is nil when the platform did not annotate the
	// flavor (unknown).
	Available *bool  `json:"available,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Image struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	MinBootSizeMiB int64  `json:"min_boot_size_mib"`
	CreatedAt      string `json:"created_at"`
}

type RegionPrice struct {
	ID           string  `json:"id"`
	RegionID     string  `json:"region_id"`
	ResourceKind string  `json:"resource_kind"`
	PricePerHour float64 `json:"price_per_hour"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type VPC struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	RegionID  string          `json:"region_id"`
	VNI       int             `json:"vni"`
	CIDR      string          `json:"cidr"`
	State     string          `json:"state"`
	Spec      json.RawMessage `json:"spec"`
	Status    json.RawMessage `json:"status"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type Volume struct {
	ID         string          `json:"id"`
	ProjectID  string          `json:"project_id"`
	RegionID   string          `json:"region_id"`
	SizeMiB    int64           `json:"size_mib"`
	CloneFrom  string          `json:"clone_from"`
	Bootable   bool            `json:"bootable"`
	State      string          `json:"state"`
	Spec       json.RawMessage `json:"spec"`
	Status     json.RawMessage `json:"status"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
	AttachedTo string          `json:"attached_to,omitempty"`
}

type NetworkInterface struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	RegionID  string          `json:"region_id"`
	VPCID     string          `json:"vpc_id"`
	IPAddress string          `json:"ip_address"`
	PrefixLen int             `json:"prefix_len"`
	State     string          `json:"state"`
	Status    json.RawMessage `json:"status"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type VM struct {
	ID                  string          `json:"id"`
	ProjectID           string          `json:"project_id"`
	RegionID            string          `json:"region_id"`
	FlavorID            string          `json:"flavor_id"`
	Class               string          `json:"class"`
	Serverless          bool            `json:"serverless"`
	Name                string          `json:"name"`
	State               string          `json:"state"`
	Status              json.RawMessage `json:"status"`
	SSHKeyIDs           []string        `json:"ssh_key_ids,omitempty"`
	VolumeIDs           []string        `json:"volume_ids,omitempty"`
	NetworkInterfaceIDs []string        `json:"network_interface_ids,omitempty"`
	Flavor              *Flavor         `json:"flavor,omitempty"`
	GatewayRoutes       []GatewayRoute  `json:"gateway_routes,omitempty"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type CreateVPCRequest struct {
	RegionID string `json:"region_id"`
	CIDR     string `json:"cidr"`
}

type PatchVPCRequest struct {
	CIDR string `json:"cidr,omitempty"`
}

type CreateVolumeRequest struct {
	RegionID string `json:"region_id"`
	SizeMiB  int64  `json:"size_mib"`
	// CloneFrom (image code) makes the volume a bootable boot disk; the API
	// requires bootable=true together with clone_from (we send both).
	CloneFrom string `json:"clone_from,omitempty"`
	Bootable  bool   `json:"bootable"`
}

type PatchVolumeRequest struct {
	SizeMiB int64 `json:"size_mib"`
}

type CreateNetworkInterfaceRequest struct {
	VPCID     string `json:"vpc_id"`
	IPAddress string `json:"ip_address,omitempty"` // optional pinned address; empty = platform assigns
}

type CreateVMRequest struct {
	RegionID            string   `json:"region_id"`
	FlavorID            string   `json:"flavor_id"`
	Class               string   `json:"class"`
	Name                string   `json:"name"`
	NetworkInterfaceIDs []string `json:"network_interface_ids"`
	SSHKeyIDs           []string `json:"ssh_key_ids"`
	VolumeIDs           []string `json:"volume_ids,omitempty"`
	BootVolumeID        string   `json:"boot_volume_id,omitempty"`
	Serverless          bool     `json:"serverless"`
}

type PatchVMRequest struct {
	Name     string `json:"name,omitempty"`
	FlavorID string `json:"flavor_id,omitempty"`
}

type ContainerImage struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	RegionID  string          `json:"region_id"`
	Ref       string          `json:"ref"`
	State     string          `json:"state"`
	Status    json.RawMessage `json:"status"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type Container struct {
	ID                  string            `json:"id"`
	ProjectID           string            `json:"project_id"`
	RegionID            string            `json:"region_id"`
	FlavorID            string            `json:"flavor_id"`
	Class               string            `json:"class"`
	Serverless          bool              `json:"serverless"`
	ImageID             string            `json:"image_id"`
	Name                string            `json:"name"`
	Command             []string          `json:"command"`
	Args                []string          `json:"args"`
	Env                 map[string]string `json:"env"`
	RestartPolicy       string            `json:"restart_policy"`
	User                string            `json:"user,omitempty"`
	State               string            `json:"state"`
	Status              json.RawMessage   `json:"status"`
	VolumeIDs           []string          `json:"volume_ids,omitempty"`
	NetworkInterfaceIDs []string          `json:"network_interface_ids,omitempty"`
	SSHKeyIDs           []string          `json:"ssh_key_ids,omitempty"`
	Flavor              *Flavor           `json:"flavor,omitempty"`
	GatewayRoutes       []GatewayRoute    `json:"gateway_routes,omitempty"`
	CreatedAt           string            `json:"created_at"`
	UpdatedAt           string            `json:"updated_at"`
}

type CreateContainerImageRequest struct {
	RegionID string `json:"region_id"`
	Ref      string `json:"ref"`
}

type WireGuardPeer struct {
	ID                  string          `json:"id"`
	ProjectID           string          `json:"project_id"`
	RegionID            string          `json:"region_id"`
	NetworkInterfaceID  string          `json:"network_interface_id"`
	PublicKey           string          `json:"public_key"`
	Comment             string          `json:"comment,omitempty"`
	PersistentKeepalive int             `json:"persistent_keepalive"`
	Enabled             bool            `json:"enabled"`
	State               string          `json:"state"`
	Status              json.RawMessage `json:"status"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

type CreateContainerRequest struct {
	RegionID            string            `json:"region_id"`
	FlavorID            string            `json:"flavor_id"`
	Class               string            `json:"class"`
	ImageID             string            `json:"image_id"`
	Name                string            `json:"name"`
	Command             []string          `json:"command"`
	Args                []string          `json:"args"`
	Env                 map[string]string `json:"env"`
	RestartPolicy       string            `json:"restart_policy"`
	User                string            `json:"user,omitempty"`
	VolumeIDs           []string          `json:"volume_ids"`
	NetworkInterfaceIDs []string          `json:"network_interface_ids,omitempty"`
	SSHKeyIDs           []string          `json:"ssh_key_ids"`
	Serverless          bool              `json:"serverless"`
}

type PatchContainerRequest struct {
	Name     *string `json:"name,omitempty"`
	FlavorID *string `json:"flavor_id,omitempty"`
	// SSHKeyIDs replaces the console-SSH allowlist (API requires >= 1 key).
	SSHKeyIDs *[]string `json:"ssh_key_ids,omitempty"`
	// Process config — updatable in place (Microsrv applies them through a
	// controlled sandbox restart; no resource replace).
	Command *[]string          `json:"command,omitempty"`
	Args    *[]string          `json:"args,omitempty"`
	Env     *map[string]string `json:"env,omitempty"`
	User    *string            `json:"user,omitempty"`
}

type CreateWireGuardPeerRequest struct {
	NetworkInterfaceID  string `json:"network_interface_id"`
	PublicKey           string `json:"public_key"`
	Comment             string `json:"comment,omitempty"`
	PersistentKeepalive *int   `json:"persistent_keepalive,omitempty"`
	Enabled             *bool  `json:"enabled,omitempty"`
}

// GatewayRoute grants console SSH/HTTP reachability for a VM or Container.
// SNIDomain is required iff Protocol is "tls" (SNI passthrough); ProxyProtocol
// is optional for tls (v1/v2 PROXY header); both immutable after create.
// TargetPort is optional iff Protocol is "http" (backend dial port, 0 =
// console default 80) and is mutable via PATCH.
type GatewayRoute struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"project_id"`
	RegionID      string          `json:"region_id"`
	ComputeID     string          `json:"compute_id"`
	Protocol      string          `json:"protocol"`
	SNIDomain     string          `json:"sni_domain,omitempty"`
	ProxyProtocol string          `json:"proxy_protocol,omitempty"`
	TargetPort    int             `json:"target_port,omitempty"`
	Comment       string          `json:"comment,omitempty"`
	Enabled       bool            `json:"enabled"`
	State         string          `json:"state"`
	Status        json.RawMessage `json:"status"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}

type CreateGatewayRouteRequest struct {
	ComputeID     string `json:"compute_id"`
	Protocol      string `json:"protocol"`
	SNIDomain     string `json:"sni_domain,omitempty"`
	ProxyProtocol string `json:"proxy_protocol,omitempty"`
	// TargetPort: http routes only, 1..65535 (absent = console default 80).
	TargetPort *int   `json:"target_port,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
}

type PatchGatewayRouteRequest struct {
	Comment *string `json:"comment,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	// TargetPort is mutable after create; 0 clears back to the console default.
	TargetPort *int `json:"target_port,omitempty"`
	// Retarget fields — updatable in place (Microsrv re-resolves them every
	// reconcile pass; the API re-validates SNI claims on each change).
	ComputeID     *string `json:"compute_id,omitempty"`
	Protocol      *string `json:"protocol,omitempty"`
	SNIDomain     *string `json:"sni_domain,omitempty"`
	ProxyProtocol *string `json:"proxy_protocol,omitempty"`
}

// LBListener maps one listen port to a target port on backends;
// protocol is tcp/udp and algorithm defaults to round_robin.
type LBListener struct {
	Protocol   string `json:"protocol"`
	ListenPort uint32 `json:"listen_port"`
	TargetPort uint32 `json:"target_port"`
	Algorithm  string `json:"algorithm,omitempty"`
}

// LBHealthCheck configures the probe engine; unset fields keep platform
// defaults.
type LBHealthCheck struct {
	Protocol           string `json:"protocol,omitempty"`
	Port               uint32 `json:"port,omitempty"`
	HTTPPath           string `json:"http_path,omitempty"`
	IntervalMS         uint32 `json:"interval_ms,omitempty"`
	TimeoutMS          uint32 `json:"timeout_ms,omitempty"`
	UnhealthyThreshold uint32 `json:"unhealthy_threshold,omitempty"`
	HealthyThreshold   uint32 `json:"healthy_threshold,omitempty"`
}

// LoadBalancer is a project-scoped L4 balancer for one VPC with an explicit
// VIP and VM/container backends.
type LoadBalancer struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	RegionID    string          `json:"region_id"`
	VPCID       string          `json:"vpc_id"`
	VIP         string          `json:"vip"`
	Listeners   []LBListener    `json:"listeners"`
	Backends    []string        `json:"backends"`
	HealthCheck LBHealthCheck   `json:"health_check"`
	State       string          `json:"state"`
	Status      json.RawMessage `json:"status"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type CreateLoadBalancerRequest struct {
	VPCID       string         `json:"vpc_id"`
	VIP         string         `json:"vip"`
	Listeners   []LBListener   `json:"listeners"`
	Backends    []string       `json:"backends"`
	HealthCheck *LBHealthCheck `json:"health_check,omitempty"`
}

type PatchLoadBalancerRequest struct {
	Listeners   *[]LBListener  `json:"listeners,omitempty"`
	Backends    *[]string      `json:"backends,omitempty"`
	HealthCheck *LBHealthCheck `json:"health_check,omitempty"`
}
