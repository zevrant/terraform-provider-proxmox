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
  peers = [data.proxmox_node.proxmox_01.network_address]
  nodes = [data.proxmox_node.proxmox_01.name]
  ipam = "pve"
}

data proxmox_node proxmox_01 {
  name = "proxmox-01"
}

data proxmox_node proxmox_02 {
  name = "proxmox-02"
}

data proxmox_node proxmox_03 {
  name = "proxmox-03"
}

resource proxmox_sdn_zone test_zone {
  type = "vxlan"
  zone = "test"
  ipam = "pve"
  nodes = [
    data.proxmox_node.proxmox_01.name,
    data.proxmox_node.proxmox_02.name,
    data.proxmox_node.proxmox_03.name
  ]
  peers = [data.proxmox_node.proxmox_01.network_address, data.proxmox_node.proxmox_02.network_address, data.proxmox_node.proxmox_03.network_address]
}