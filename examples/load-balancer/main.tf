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

data "microsrv_flavor" "std" {
  code = "std.1c2g"
}

data "microsrv_image" "ubuntu" {
  code = "ubuntu-noble"
}

resource "microsrv_project" "demo" {
  name = "microsrv-lb"
}

resource "microsrv_ssh_key" "admin" {
  name       = "admin"
  public_key = var.ssh_public_key
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.45.0.0/24"
}

resource "microsrv_network_interface" "eth0" {
  for_each   = toset(["a", "b"])
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

# One explicit boot disk per VM (a boot volume attaches to a single VM).
resource "microsrv_volume" "boot" {
  for_each   = toset(["a", "b"])
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 20480
  clone_from = data.microsrv_image.ubuntu.code
}

resource "microsrv_vm" "app" {
  for_each              = toset(["a", "b"])
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.std.id
  class                 = "shared"
  name                  = "app-${each.key}"
  network_interface_ids = [microsrv_network_interface.eth0[each.key].id]
  ssh_key_ids           = [microsrv_ssh_key.admin.id]
  boot_volume_id        = microsrv_volume.boot[each.key].id
}

resource "microsrv_load_balancer" "web" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
  vip        = "10.45.0.10"

  listeners = [
    {
      protocol    = "tcp"
      listen_port = 80
      target_port = 80
    },
    {
      protocol    = "tcp"
      listen_port = 22
      target_port = 22
      algorithm   = "source_ip"
    },
  ]

  backends = [for vm in microsrv_vm.app : vm.id]

  health_check = {
    protocol    = "tcp"
    port        = 80
    interval_ms = 1000
    timeout_ms  = 500
  }
}

output "lb_id" {
  value = microsrv_load_balancer.web.id
}

output "lb_state" {
  value = microsrv_load_balancer.web.state
}

variable "ssh_public_key" {
  type = string
}
