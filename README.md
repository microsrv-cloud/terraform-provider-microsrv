# Terraform Provider Microsrv

Terraform/OpenTofu provider (`microsrv`) for managed infrastructure via the Microsrv control plane (`https://api.microsrv.ru`).

## Provider

```hcl
terraform {
  required_providers {
    microsrv = {
      source  = "microsrv-cloud/microsrv"
      version = "0.1.0"
    }
  }
}

provider "microsrv" {
  endpoint = "https://api.microsrv.ru"
  token    = var.microsrv_token
}
```

Auth: a Microsrv **access token** (`sel_…`, the only supported credential for Terraform/CI). Mint one in the console under *Access → Tokens* or with `POST /api/v1/tokens {"name":"terraform"}` — the plaintext is shown once. Pass it as `token` (or the `api_key` alias).

Env: `MICROSRV_ENDPOINT`, `MICROSRV_TOKEN` (or `MICROSRV_API_KEY`, mutually exclusive with `MICROSRV_TOKEN`).

## Install (OpenTofu OCI mirror)

Every version tag (`v*.*.*`) publishes the provider to GHCR as an OCI artifact (OpenTofu ≥ 1.7):

```
ghcr.io/microsrv-cloud/terraform-provider-microsrv:<version>
```

Until it lands in the official registry, point OpenTofu at GHCR in `~/.tofurc` (or the file named by `TF_CLI_CONFIG_FILE`):

```hcl
provider_installation {
  oci_mirror {
    repository_template = "ghcr.io/microsrv-cloud/terraform-provider-microsrv"
    include             = ["registry.opentofu.org/microsrv-cloud/microsrv"]
  }
  direct {
    exclude = ["registry.opentofu.org/microsrv-cloud/microsrv"]
  }
}
```

Then use the provider normally (`source = "microsrv-cloud/microsrv"`). Anonymous pulls from ghcr.io need OpenTofu ≥ 1.13 — v1.12.x fails against ghcr's `/v2/` challenge ([opentofu#3316](https://github.com/opentofu/opentofu/issues/3316)).

Note: custom installation methods record lock-file checksums for the current platform only; run `tofu providers lock -platform=…` once per platform your team uses.

## Build

```bash
make build
make test
make install   # installs to ~/.terraform.d/plugins/.../microsrv-cloud/microsrv/dev/...
```

## Data sources

| Data source | Notes |
| ---------- | ------- |
| `microsrv_regions` / `microsrv_region` | List all, or look up by `code` |
| `microsrv_flavors` / `microsrv_flavor` | List all (`code`, `vcpus`, `memory_mib`, `available`), or look up by `code` |
| `microsrv_images` / `microsrv_image` | List all, or look up by `code` |
| `microsrv_quotas`, `microsrv_pricing` | Project quotas / pricing |
| `microsrv_project` | Look up by `id` or `name` |

```hcl
data "microsrv_regions" "all" {}

data "microsrv_flavors" "all" {}

locals {
  region = data.microsrv_regions.all.regions[0].code
  flavor = one([for f in data.microsrv_flavors.all.flavors : f.code if f.code == "std" && f.available])
}
```

## Resources

| Resource | Notes |
| ---------- | ------- |
| `microsrv_project` | ForceNew on name |
| `microsrv_ssh_key` | ForceNew |
| `microsrv_vpc` | Waits READY; `/24` CIDR |
| `microsrv_volume` | Data disk (blank) or boot disk (`clone_from` ⇒ bootable, immutable); 512 MiB quantum; `attached_to` shows owner when attached |
| `microsrv_network_interface` | IP assigned by the platform |
| `microsrv_vm` | Waits READY or CHECKPOINTED; `class` required; boot disk is an explicit `microsrv_volume` via required `boot_volume_id` (no inline boot); `serverless` (scale-to-zero, immutable); destroy detaches and leaves volumes to their own resources; VM flavors require ≥1 vCPU / 2 GiB (micro flavors are container-only) |
| `microsrv_container_image` | OCI image pull; Waits READY; ForceNew |
| `microsrv_container` | gVisor container; `class` required; `serverless` scale-to-zero; `user` override (`root` or image user); `command` may be empty (inherits image entrypoint/cmd); `ssh_key_ids` optional (unset = project keyring, updatable 1..N); `name`/`flavor_id`/`ssh_key_ids` updatable; waits READY or CHECKPOINTED |
| `microsrv_gateway_route` | Console SSH/HTTP route or TLS SNI passthrough (`sni_domain`) for a VM/container/LB; `proxy_protocol` v1/v2 for tls; `target_port` backend dial port for http (mutable); `comment`/`enabled`/`target_port` updatable |
| `microsrv_load_balancer` | L4 balancer with explicit VPC VIP; `listeners`/`backends`/`health_check` updatable |
| `microsrv_wireguard_peer` | WireGuard client peer on a network interface; public key only; ForceNew |

## Examples

See [`examples/basic`](examples/basic), [`examples/with-data-volume`](examples/with-data-volume), [`examples/container`](examples/container), [`examples/load-balancer`](examples/load-balancer), and [`examples/wireguard-peer`](examples/wireguard-peer).
