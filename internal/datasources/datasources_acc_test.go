package datasources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/acctest"
)

// ---------------------------------------------------------------------------
// microsrv_regions (list)
// ---------------------------------------------------------------------------

func TestAccDataRegions_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_regions" "all" {}
`,
				Check: resource.ComposeTestCheckFunc(
					// There must be at least one region.
					resource.TestCheckResourceAttrSet("data.microsrv_regions.all", "regions.0.id"),
					resource.TestCheckResourceAttrSet("data.microsrv_regions.all", "regions.0.code"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_region (single lookup by code)
// ---------------------------------------------------------------------------

func TestAccDataRegion_byCode(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Use microsrv_regions to find the first enabled region code, then
				// do a single-region lookup by that code.  This avoids hard-coding
				// the region code.
				Config: acctest.ProviderConfig + `
data "microsrv_regions" "all" {}

locals {
  first_code = data.microsrv_regions.all.regions[0].code
}

data "microsrv_region" "first" {
  code = local.first_code
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_region.first", "id"),
					resource.TestCheckResourceAttrSet("data.microsrv_region.first", "code"),
					resource.TestCheckResourceAttrSet("data.microsrv_region.first", "name"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_flavors (list) + microsrv_flavor (single)
// ---------------------------------------------------------------------------

func TestAccDataFlavors_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_flavors" "all" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_flavors.all", "flavors.0.id"),
					resource.TestCheckResourceAttrSet("data.microsrv_flavors.all", "flavors.0.code"),
				),
			},
		},
	})
}

func TestAccDataFlavor_byCode(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_flavors" "all" {}

locals {
  first_flavor_code = data.microsrv_flavors.all.flavors[0].code
}

data "microsrv_flavor" "first" {
  code = local.first_flavor_code
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_flavor.first", "id"),
					resource.TestCheckResourceAttrSet("data.microsrv_flavor.first", "code"),
					resource.TestCheckResourceAttrSet("data.microsrv_flavor.first", "vcpus"),
					resource.TestCheckResourceAttrSet("data.microsrv_flavor.first", "memory_mib"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_images (list) + microsrv_image (single)
// ---------------------------------------------------------------------------

func TestAccDataImages_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_images" "all" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_images.all", "images.0.id"),
					resource.TestCheckResourceAttrSet("data.microsrv_images.all", "images.0.code"),
				),
			},
		},
	})
}

func TestAccDataImage_byCode(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_images" "all" {}

locals {
  first_image_code = data.microsrv_images.all.images[0].code
}

data "microsrv_image" "first" {
  code = local.first_image_code
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_image.first", "id"),
					resource.TestCheckResourceAttrSet("data.microsrv_image.first", "code"),
					resource.TestCheckResourceAttrSet("data.microsrv_image.first", "name"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_quotas
// ---------------------------------------------------------------------------

func TestAccDataQuotas_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_quotas" "me" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_quotas.me", "quotas.0.id"),
					resource.TestCheckResourceAttrSet("data.microsrv_quotas.me", "quotas.0.resource_kind"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_pricing
// ---------------------------------------------------------------------------

func TestAccDataPricing_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + `
data "microsrv_regions" "all" {}
data "microsrv_pricing" "r" {
  region_id = data.microsrv_regions.all.regions[0].id
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.microsrv_pricing.r", "prices.0.id"),
					resource.TestCheckResourceAttrSet("data.microsrv_pricing.r", "prices.0.resource_kind"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// microsrv_project (data source)
// ---------------------------------------------------------------------------

func TestAccDataProject_basic(t *testing.T) {
	acctest.SkipIfNoAccEnv(t)
	suffix := acctest.RandSuffix()
	name := fmt.Sprintf("tf-acc-ds-proj-%s", suffix)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig + fmt.Sprintf(`
resource "microsrv_project" "p" { name = %q }

data "microsrv_project" "lookup" {
  id = microsrv_project.p.id
}
`, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.microsrv_project.lookup", "id",
						"microsrv_project.p", "id",
					),
					resource.TestCheckResourceAttr("data.microsrv_project.lookup", "name", name),
				),
			},
		},
	})
}
