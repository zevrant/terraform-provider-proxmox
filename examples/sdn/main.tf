terraform {
  required_providers {
    proxmox = {
      source = "app.terraform.io/zevrant-services/proxmox"
    }
  }

}

provider "proxmox" {
  verify_tls = false
  host       = "https://10.0.0.2:8006"
  username   = var.proxmox_username
  password   = var.proxmox_password
}

data proxmox_sdn_zone test {
  zone = "core"
}

data proxmox_node proxmox_01 {
  name = "proxmox-01"
}

resource proxmox_sdn_zone test {
  zone = "tfTest2"
  type = "vxlan"
  peers = [data.proxmox_node.proxmox_01.network_address]
  nodes = [data.proxmox_node.proxmox_01.name]
  ipam = "pve"
}