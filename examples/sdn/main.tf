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

resource proxmox_sdn_zone test {
  zone = "tfTest2"
  type = "vxlan"
  peers = ["10.0.0.2"]
  nodes = ["proxmox-01"]
  ipam = "pve"
}