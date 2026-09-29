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
  name = "microsrv-data-vol"
}

resource "microsrv_ssh_key" "admin" {
  name       = "admin"
  public_key = var.ssh_public_key
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.43.0.0/24"
}

resource "microsrv_network_interface" "eth0" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

resource "microsrv_volume" "data" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 10240
}

# Boot disk is an explicit resource (clone_from => bootable).
resource "microsrv_volume" "boot" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  size_mib   = 20480
  clone_from = data.microsrv_image.ubuntu.code
}

resource "microsrv_vm" "app" {
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.std.id
  class                 = "shared"
  name                  = "app"
  network_interface_ids = [microsrv_network_interface.eth0.id]
  ssh_key_ids           = [microsrv_ssh_key.admin.id]
  volume_ids            = [microsrv_volume.data.id]
  boot_volume_id        = microsrv_volume.boot.id
}

variable "ssh_public_key" {
  type = string
}
