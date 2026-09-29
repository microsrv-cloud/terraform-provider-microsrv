# gVisor container example: gateway reachability (if needed) is a dedicated
# microsrv_gateway_route resource, not inline expose_* flags.
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

resource "microsrv_project" "demo" {
  name = "microsrv-container"
}

resource "microsrv_vpc" "main" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  cidr       = "10.44.0.0/24"
}

resource "microsrv_ssh_key" "admin" {
  name       = "admin"
  public_key = var.ssh_public_key
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

resource "microsrv_container_image" "nginx" {
  project_id = microsrv_project.demo.id
  region_id  = data.microsrv_region.yc.id
  ref        = "nginx:1.27"
}

resource "microsrv_container" "web" {
  project_id            = microsrv_project.demo.id
  region_id             = data.microsrv_region.yc.id
  flavor_id             = data.microsrv_flavor.std.id
  class                 = "shared"
  image_id              = microsrv_container_image.nginx.id
  name                  = "web"
  command               = ["nginx", "-g"]
  args                  = ["daemon off;"]
  env                   = { NGINX_ENTRYPOINT_QUIET_LOGS = "1" }
  restart_policy        = "always"
  volume_ids            = [microsrv_volume.data.id]
  network_interface_ids = [microsrv_network_interface.eth0.id]
  ssh_key_ids           = [microsrv_ssh_key.admin.id]
}

# HTTP gateway as a dedicated route: the console gateway dials the container's
# backend port (target_port 80 = nginx; omit for the console default 80).
# Implicit dependency on the container via compute_id — the route is created
# only after the container is READY.
resource "microsrv_gateway_route" "web_http" {
  project_id  = microsrv_project.demo.id
  compute_id  = microsrv_container.web.id
  protocol    = "http"
  target_port = 80
  comment     = "nginx http"
}

output "container_id" {
  value = microsrv_container.web.id
}

output "gateway_route_id" {
  value = microsrv_gateway_route.web_http.id
}

variable "ssh_public_key" {
  type = string
}
