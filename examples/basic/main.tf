terraform {
  required_providers {
    microsrv = {
      source = "microsrv-cloud/microsrv"
    }
  }
}

provider "microsrv" {
  # endpoint / token via MICROSRV_ENDPOINT and MICROSRV_TOKEN
}

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
  name = "microsrv-demo"
}

resource "microsrv_ssh_key" "admin" {
  name       = "admin"
  public_key = var.ssh_public_key
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.42.0.0/24"
}

resource "microsrv_network_interface" "eth0" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

# Boot disk is an explicit resource (clone_from => bootable), never created
# inline by the VM.
resource "microsrv_volume" "boot" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 20480
  clone_from = data.microsrv_image.ubuntu.code
}

resource "microsrv_vm" "web" {
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.std.id
  class                 = "shared"
  name                  = "web"
  network_interface_ids = [microsrv_network_interface.eth0.id]
  ssh_key_ids           = [microsrv_ssh_key.admin.id]
  boot_volume_id        = microsrv_volume.boot.id
}

# Console reachability is a dedicated gateway route, not inline expose_* flags.
resource "microsrv_gateway_route" "web_ssh" {
  project_id = microsrv_project.demo.id
  compute_id = microsrv_vm.web.id
  protocol   = "ssh"
}

variable "ssh_public_key" {
  type = string
}

output "vm_id" {
  value = microsrv_vm.web.id
}

output "ni_ip" {
  value = microsrv_network_interface.eth0.ip_address
}
