terraform {
  required_providers {
    microsrv = {
      source = "microsrv-cloud/microsrv"
    }
  }
}

provider "microsrv" {}

data "microsrv_region" "yc" {
  code = "ru-yndx-central"
}

# Micro flavor for container-only workload (512 MiB)
data "microsrv_flavor" "micro" {
  code = "micro.1c512m"
}

# Standard flavor for VM (serverless still needs ≥1c2g)
data "microsrv_flavor" "std" {
  code = "std.1c2g"
}

data "microsrv_image" "ubuntu" {
  code = "ubuntu-noble"
}

resource "microsrv_project" "demo" {
  name = "microsrv-serverless"
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.45.0.0/24"
}

resource "microsrv_ssh_key" "admin" {
  name       = "admin"
  public_key = var.ssh_public_key
}

resource "microsrv_network_interface" "vm_ni" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

resource "microsrv_network_interface" "ctr_ni" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

resource "microsrv_volume" "ctr_data" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 512
}

resource "microsrv_container_image" "echo" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  ref        = "ghcr.io/microsrv/echo:latest"
}

# Boot disk is an explicit resource (clone_from => bootable).
resource "microsrv_volume" "boot" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 8192
  clone_from = data.microsrv_image.ubuntu.code
}

# Serverless VM (checkpointed on idle, spec 66). State may be CHECKPOINTED.
resource "microsrv_vm" "idle_web" {
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.std.id
  class                 = "shared"
  serverless            = true
  name                  = "idle-web"
  network_interface_ids = [microsrv_network_interface.vm_ni.id]
  ssh_key_ids           = [microsrv_ssh_key.admin.id]
  boot_volume_id        = microsrv_volume.boot.id
}

# Serverless micro container with user override and empty command inheritance demo.
resource "microsrv_container" "echo" {
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.micro.id
  class                 = "shared"
  serverless            = true
  image_id              = microsrv_container_image.echo.id
  name                  = "echo"
  command               = [] # inherit from image
  volume_ids            = [microsrv_volume.ctr_data.id]
  network_interface_ids = [microsrv_network_interface.ctr_ni.id]
  ssh_key_ids           = [] # no console SSH
  user                  = "root"
}

# TLS SNI passthrough with PROXY protocol v2 (spec 64)
resource "microsrv_gateway_route" "echo_tls" {
  project_id     = microsrv_project.demo.id
  compute_id     = microsrv_container.echo.id
  protocol       = "tls"
  sni_domain     = "echo.example.com"
  proxy_protocol = "v2"
}

variable "ssh_public_key" {
  type = string
}
