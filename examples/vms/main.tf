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

data proxmox_vm test {
  node_name = "proxmox-01"
  vm_id = "110"
}

resource proxmox_vm test {
  name = "terraform-test-vm-nextcloud-clone"
  qemu_agent_enabled = true
  cores = "7"
  memory = "16384"
  os_type = "l26"
  description = "terraform testing vm"
  node_name = "proxmox-01"
  vm_id = "9001"
  cpu_type = "host"
  boot_order = ["scsi0"]
  auto_start = false
  host_startup_order = 1
  protection = false
  nameserver = "192.168.0.1"
  ip_configs { #TODO: make singular
    ip_address = "10.0.0.222/24"
    gateway = "10.0.0.1"
    order = 0
  }

  disks { #TODO: make singular
    bus_type = "scsi"
    storage_location = "exosDisks"
    size = "50G"
    order = 0
  }

  disks {
    bus_type = "scsi"
    storage_location = "sataLarge"
    size = "1002G"
    order = 1
  }

  disks {
    bus_type = "scsi"
    storage_location = "exosDisks"
    size = "1000G"
    order = 2
  }

  disks {
    bus_type = "scsi"
    storage_location = "exosDisks"
    size = "2G"
    order = 3
  }
  disks {
    bus_type = "scsi"
    storage_location = "exosDisks"
    size = "1G"
    order = 4
  }

  network_interfaces { #TODO: make singular
    mac_address = "1a:2b:3c:4e:5f:60"
    bridge = "vmbr0"
    firewall = true
    order = 0
  }
}