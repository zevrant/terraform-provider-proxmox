terraform {
  required_version = "~> 1.5"
  required_providers {
    http-client = {
      source  = "dmachard/http-client"
      version = "0.3.0"
    }
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