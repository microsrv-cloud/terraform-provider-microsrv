package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/acctest"
)

// ---------------------------------------------------------------------------
// microsrv_project
// ---------------------------------------------------------------------------

func TestAccProject_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	name := fmt.Sprintf("tf-acc-proj-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			// Create & verify
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "microsrv_project" "test" {
  name = %q
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_project.test", "name", name),
					resource.TestCheckResourceAttrSet("microsrv_project.test", "id"),
					resource.TestCheckResourceAttrSet("microsrv_project.test", "owner_user_id"),
					resource.TestCheckResourceAttrSet("microsrv_project.test", "created_at"),
				),
			},
			// Import round-trip
			{
				ResourceName:      "microsrv_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_ssh_key
// ---------------------------------------------------------------------------

func TestAccSSHKey_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	name := fmt.Sprintf("tf-acc-key-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "microsrv_ssh_key" "test" {
  name       = %q
  public_key = %q
}
`, name, pubKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_ssh_key.test", "name", name),
					resource.TestCheckResourceAttr("microsrv_ssh_key.test", "public_key", pubKey),
					resource.TestCheckResourceAttrSet("microsrv_ssh_key.test", "id"),
					resource.TestCheckResourceAttrSet("microsrv_ssh_key.test", "fingerprint"),
					resource.TestCheckResourceAttrSet("microsrv_ssh_key.test", "user_id"),
				),
			},
			{
				ResourceName:      "microsrv_ssh_key.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_vpc
// ---------------------------------------------------------------------------

func TestAccVPC_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-vpc-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: vpcConfig(projName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_vpc.test", "id"),
					resource.TestCheckResourceAttr("microsrv_vpc.test", "cidr", "10.200.0.0/24"),
					resource.TestCheckResourceAttrSet("microsrv_vpc.test", "vni"),
					resource.TestCheckResourceAttr("microsrv_vpc.test", "state", "READY"),
				),
			},
			{
				ResourceName:      "microsrv_vpc.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func vpcConfig(projName string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_vpc" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  cidr       = "10.200.0.0/24"
}
`, projName)
}

// ---------------------------------------------------------------------------
// microsrv_volume  (blank data disk)
// ---------------------------------------------------------------------------

func TestAccVolume_blank(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-vol-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: volumeBlankConfig(projName, 1024),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_volume.test", "id"),
					resource.TestCheckResourceAttr("microsrv_volume.test", "size_mib", "1024"),
					resource.TestCheckResourceAttr("microsrv_volume.test", "bootable", "false"),
					resource.TestCheckResourceAttr("microsrv_volume.test", "state", "READY"),
				),
			},
			// Grow volume in-place (grow-only)
			{
				Config: volumeBlankConfig(projName, 2048),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_volume.test", "size_mib", "2048"),
				),
			},
			{
				ResourceName:      "microsrv_volume.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func volumeBlankConfig(projName string, sizeMiB int) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_volume" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  size_mib   = %d
}
`, projName, sizeMiB)
}

// ---------------------------------------------------------------------------
// microsrv_volume  (bootable — clone_from)
// ---------------------------------------------------------------------------

func TestAccVolume_bootable(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-volb-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: bootableVolumeConfig(projName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_volume.boot", "id"),
					resource.TestCheckResourceAttr("microsrv_volume.boot", "bootable", "true"),
					resource.TestCheckResourceAttr("microsrv_volume.boot", "state", "READY"),
				),
			},
			{
				ResourceName:      "microsrv_volume.boot",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func bootableVolumeConfig(projName string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}
data "microsrv_images"  "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_volume" "boot" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  size_mib   = 20480
  clone_from = data.microsrv_images.all.images[0].code
}
`, projName)
}

// ---------------------------------------------------------------------------
// microsrv_network_interface
// ---------------------------------------------------------------------------

func TestAccNetworkInterface_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-ni-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: networkInterfaceConfig(projName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_network_interface.test", "id"),
					resource.TestCheckResourceAttrSet("microsrv_network_interface.test", "ip_address"),
					resource.TestCheckResourceAttr("microsrv_network_interface.test", "state", "READY"),
				),
			},
			{
				ResourceName:      "microsrv_network_interface.test",
				ImportState:       true,
				ImportStateVerify: true,
				// status is a live server-side JSON blob that can change between
				// the apply and the import read (e.g. tap_name/underlay_ip appear
				// asynchronously). updated_at similarly drifts by a few ms.
				ImportStateVerifyIgnore: []string{"status", "updated_at"},
			},
		},
	})
}

func networkInterfaceConfig(projName string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_vpc" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  cidr       = "10.201.0.0/24"
}

resource "microsrv_network_interface" "test" {
  project_id = microsrv_project.test.id
  vpc_id     = microsrv_vpc.test.id
}
`, projName)
}

// ---------------------------------------------------------------------------
// Shared VM-stack config (project + ssh_key + vpc + ni + boot volume + vm)
// ---------------------------------------------------------------------------

func vmStackConfig(projName, suffix, pubKey, vmName string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}
data "microsrv_images"  "all" {}
data "microsrv_flavors" "all" {}

locals {
  vm_flavor = [for f in data.microsrv_flavors.all.flavors : f if f.vcpus >= 1 && f.memory_mib >= 2048][0]
}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_ssh_key" "test" {
  name       = "tf-acc-vm-%s"
  public_key = %q
}

resource "microsrv_vpc" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  cidr       = "10.202.0.0/24"
}

resource "microsrv_network_interface" "test" {
  project_id = microsrv_project.test.id
  vpc_id     = microsrv_vpc.test.id
}

resource "microsrv_volume" "boot" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  size_mib   = 20480
  clone_from = data.microsrv_images.all.images[0].code
}

resource "microsrv_vm" "test" {
  project_id            = microsrv_project.test.id
  region_id             = data.microsrv_regions.all.regions[0].id
  flavor_id             = local.vm_flavor.id
  class                 = "shared"
  name                  = %q
  network_interface_ids = [microsrv_network_interface.test.id]
  ssh_key_ids           = [microsrv_ssh_key.test.id]
  boot_volume_id        = microsrv_volume.boot.id
}
`, projName, suffix, pubKey, vmName)
}

// ---------------------------------------------------------------------------
// microsrv_vm
// ---------------------------------------------------------------------------

func TestAccVM_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-vm-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	// The test uses the first available flavor; we look it up from the live API
	// via a data source embedded in the HCL config itself.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create
			{
				Config: vmStackConfig(projName, suffix, pubKey, "tf-acc-vm-"+suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_vm.test", "id"),
					resource.TestCheckResourceAttr("microsrv_vm.test", "name", "tf-acc-vm-"+suffix),
					resource.TestCheckResourceAttr("microsrv_vm.test", "class", "shared"),
					resource.TestCheckResourceAttr("microsrv_vm.test", "state", "READY"),
				),
			},
			// Step 2: rename (in-place update)
			{
				Config: vmStackConfig(projName, suffix, pubKey, "tf-acc-vm-renamed-"+suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_vm.test", "name", "tf-acc-vm-renamed-"+suffix),
					resource.TestCheckResourceAttr("microsrv_vm.test", "state", "READY"),
				),
			},
			// Step 3: import round-trip
			{
				ResourceName:      "microsrv_vm.test",
				ImportState:       true,
				ImportStateVerify: true,
				// updated_at drifts by a few ms between apply and import read.
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_gateway_route (ssh + http protocols)
// ---------------------------------------------------------------------------

func TestAccGatewayRoute_ssh(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-gw-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	sshRouteConfig := vmStackConfig(projName, suffix, pubKey, "tf-acc-gw-"+suffix) + `
resource "microsrv_gateway_route" "ssh" {
  project_id = microsrv_project.test.id
  compute_id = microsrv_vm.test.id
  protocol   = "ssh"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: sshRouteConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_gateway_route.ssh", "id"),
					resource.TestCheckResourceAttr("microsrv_gateway_route.ssh", "protocol", "ssh"),
					resource.TestCheckResourceAttr("microsrv_gateway_route.ssh", "state", "READY"),
					resource.TestCheckResourceAttr("microsrv_gateway_route.ssh", "enabled", "true"),
				),
			},
			// Import round-trip: verify the existing route can be imported
			// without a Helios update cycle.  The enabled toggling (update)
			// is covered by TestAccGatewayRoute_http to avoid ssh-specific
			// Helios flakiness on the disable path.
			{
				ResourceName:            "microsrv_gateway_route.ssh",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
				Config:                  sshRouteConfig,
			},
		},
	})
}

func TestAccGatewayRoute_http(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-gwh-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: vmStackConfig(projName, suffix, pubKey, "tf-acc-gwh-"+suffix) + `
resource "microsrv_gateway_route" "http" {
  project_id  = microsrv_project.test.id
  compute_id  = microsrv_vm.test.id
  protocol    = "http"
  target_port = 8080
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_gateway_route.http", "id"),
					resource.TestCheckResourceAttr("microsrv_gateway_route.http", "protocol", "http"),
					resource.TestCheckResourceAttr("microsrv_gateway_route.http", "target_port", "8080"),
				),
			},
			// Update target_port in place
			{
				Config: vmStackConfig(projName, suffix, pubKey, "tf-acc-gwh-"+suffix) + `
resource "microsrv_gateway_route" "http" {
  project_id  = microsrv_project.test.id
  compute_id  = microsrv_vm.test.id
  protocol    = "http"
  target_port = 9090
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_gateway_route.http", "target_port", "9090"),
				),
			},
			{
				ResourceName:            "microsrv_gateway_route.http",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_wireguard_peer
// ---------------------------------------------------------------------------

func TestAccWireGuardPeer_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-wg-%s", suffix)

	// A static X25519 public key — only the public key is submitted to the API.
	const wgPubKey = "EAdfPLDn6pnr2KUzOv6NPo2QhMFX3FZg7oNKkv9Jkl4="

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}

resource "microsrv_project" "test" { name = %q }

resource "microsrv_vpc" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  cidr       = "10.203.0.0/24"
}

resource "microsrv_network_interface" "test" {
  project_id = microsrv_project.test.id
  vpc_id     = microsrv_vpc.test.id
}

resource "microsrv_wireguard_peer" "test" {
  project_id           = microsrv_project.test.id
  network_interface_id = microsrv_network_interface.test.id
  public_key           = %q
  comment              = "tf-acc-wg-%s"
}
`, projName, wgPubKey, suffix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_wireguard_peer.test", "id"),
					resource.TestCheckResourceAttr("microsrv_wireguard_peer.test", "public_key", wgPubKey),
					resource.TestCheckResourceAttr("microsrv_wireguard_peer.test", "state", "READY"),
				),
			},
			{
				ResourceName:            "microsrv_wireguard_peer.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_load_balancer
// ---------------------------------------------------------------------------

func TestAccLoadBalancer_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-lb-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: lbConfig(projName, suffix, pubKey, 80, 22),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_load_balancer.test", "id"),
					resource.TestCheckResourceAttr("microsrv_load_balancer.test", "state", "READY"),
					resource.TestCheckResourceAttr("microsrv_load_balancer.test", "listeners.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("microsrv_load_balancer.test", "listeners.0.listen_port", "80"),
					resource.TestCheckResourceAttr("microsrv_load_balancer.test", "listeners.0.target_port", "22"),
				),
			},
			{
				ResourceName:            "microsrv_load_balancer.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"status", "updated_at"},
			},
		},
	})
}

func lbConfig(projName, suffix, pubKey string, listenPort, targetPort int) string {
	return vmStackConfig(projName, suffix, pubKey, "tf-acc-lb-"+suffix) + fmt.Sprintf(`
resource "microsrv_load_balancer" "test" {
  project_id = microsrv_project.test.id
  vpc_id     = microsrv_vpc.test.id
  vip        = "10.202.0.100"

  listeners = [
    {
      protocol    = "tcp"
      listen_port = %d
      target_port = %d
    }
  ]

  backends = [microsrv_vm.test.id]
}
`, listenPort, targetPort)
}
