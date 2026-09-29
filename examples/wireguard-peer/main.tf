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

# Generate a keypair locally: wg genkey | tee privatekey | wg pubkey > publickey
variable "wireguard_public_key" {
  description = "Base64 X25519 public key (32 bytes) of the local WireGuard keypair."
  type        = string
}

resource "microsrv_project" "demo" {
  name = "microsrv-wireguard"
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.45.0.0/24"
}

resource "microsrv_network_interface" "eth0" {
  project_id = microsrv_project.demo.id
  vpc_id     = microsrv_vpc.main.id
}

resource "microsrv_wireguard_peer" "laptop" {
  project_id           = microsrv_project.demo.id
  network_interface_id = microsrv_network_interface.eth0.id
  public_key           = var.wireguard_public_key
  comment              = "laptop"
}

output "peer_id" {
  value = microsrv_wireguard_peer.laptop.id
}

# Once READY, status carries allocated_ip, endpoint_addr and server_public_key
# for building the local wg0.conf.
output "peer_status" {
  value = microsrv_wireguard_peer.laptop.status
}
