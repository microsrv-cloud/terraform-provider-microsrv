package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/acctest"
)

// containerImageConfig builds the HCL for a microsrv_container_image.
// The image ref is mirror.gcr.io/library/nginx:latest (a public, auth-free mirror).
func containerImageConfig(projName string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_container_image" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  ref        = "mirror.gcr.io/library/nginx:latest"
}
`, projName)
}

// ---------------------------------------------------------------------------
// microsrv_container_image
// ---------------------------------------------------------------------------

func TestAccContainerImage_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-img-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: containerImageConfig(projName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_container_image.test", "id"),
					resource.TestCheckResourceAttr("microsrv_container_image.test", "ref", "mirror.gcr.io/library/nginx:latest"),
					resource.TestCheckResourceAttr("microsrv_container_image.test", "state", "READY"),
				),
			},
			{
				ResourceName:            "microsrv_container_image.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ref", "state", "status", "updated_at"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_container  (full lifecycle: create, env update, import)
// ---------------------------------------------------------------------------

func containerStackConfig(projName, suffix, pubKey, envVal string) string {
	return acctest.ProviderConfig + fmt.Sprintf(`
data "microsrv_regions" "all" {}
data "microsrv_flavors" "all" {}

resource "microsrv_project" "test" {
  name = %q
}

resource "microsrv_ssh_key" "test" {
  name       = "tf-acc-ct-%s"
  public_key = %q
}

resource "microsrv_vpc" "test" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  cidr       = "10.204.0.0/24"
}

resource "microsrv_network_interface" "test" {
  project_id = microsrv_project.test.id
  vpc_id     = microsrv_vpc.test.id
}

# Data volume (container requires >= 1 volume)
resource "microsrv_volume" "data" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  size_mib   = 1024
}

resource "microsrv_container_image" "nginx" {
  project_id = microsrv_project.test.id
  region_id  = data.microsrv_regions.all.regions[0].id
  ref        = "mirror.gcr.io/library/nginx:latest"
}

resource "microsrv_container" "test" {
  project_id           = microsrv_project.test.id
  region_id            = data.microsrv_regions.all.regions[0].id
  flavor_id            = data.microsrv_flavors.all.flavors[0].id
  class                = "shared"
  image_id             = microsrv_container_image.nginx.id
  name                 = "tf-acc-ct-%s"
  restart_policy       = "always"
  volume_ids           = [microsrv_volume.data.id]
  network_interface_ids = [microsrv_network_interface.test.id]
  ssh_key_ids          = [microsrv_ssh_key.test.id]

  env = {
    ACC_TEST_VAR = %q
  }
}
`, projName, suffix, pubKey, suffix, envVal)
}

func TestAccContainer_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	projName := fmt.Sprintf("tf-acc-ct-%s", suffix)
	pubKey := acctest.RandSSHPublicKey()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create
			{
				Config: containerStackConfig(projName, suffix, pubKey, "initial"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("microsrv_container.test", "id"),
					resource.TestCheckResourceAttr("microsrv_container.test", "state", "READY"),
					resource.TestCheckResourceAttr("microsrv_container.test", "restart_policy", "always"),
					resource.TestCheckResourceAttr("microsrv_container.test", "env.ACC_TEST_VAR", "initial"),
				),
			},
			// Step 2: update env in-place (no replace)
			{
				Config: containerStackConfig(projName, suffix, pubKey, "updated"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("microsrv_container.test", "env.ACC_TEST_VAR", "updated"),
					resource.TestCheckResourceAttr("microsrv_container.test", "state", "READY"),
				),
			},
			// Step 3: import round-trip
			{
				ResourceName:            "microsrv_container.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: acctest.VolatileImportAttrs,
			},
		},
	})
}
