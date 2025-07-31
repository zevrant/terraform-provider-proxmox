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
  name = "terraform-test-vm"
  qemu_agent_enabled = true
  cores = "2"
  memory = "4096"
  os_type = "l26"
  description = "terraform testing vm"
  node_name = "proxmox-03"
  vm_id = "9001"
  cpu_type = "host"
  boot_order = ["scsi0"]
  host_startup_order = 1
  protection = false
  nameserver = "10.1.0.123"
  start_on_boot = true
  default_user = "zevrant"
  cloud_init_storage_name = "exosDisks"
  power_state = "running"
  ssh_keys = [
    "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAIbmlzdHAzODQAAABhBLtOxtriPtNmisKkmfHfCByaTYCHRsDHyzQAi0yL6LUeKybjYExfR6N0xBMcIj6M/b5U3aafjKayX4nMvV7s7/vcrpBfW+WvxOCBWTlhKGNpUmAS9ApFDn51/FTuRgB/YA=="
  ]
  ip_config {
    ip_address = "10.1.0.100/24"
    gateway = "10.1.0.1"
    order = 0
  }

  disk {
    bus_type = "scsi"
    storage_location = "local-zfs"
    size = "50G"
    order = 0
    import_from = "local"
    //Must be preloaded at this location, full path is /var/lib/vz/images/0/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2
    //Long term recommendation is to use an nfs mount or something that supports RWM
    import_path = "0/alma-base-image-0.0.16.qcow2"
  }

  disk {
    bus_type = "scsi"
    storage_location = "local-zfs"
    size = "50G"
    order = 1
    import_from = "local"
    //Must be preloaded at this location, full path is /var/lib/vz/images/0/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2
    //Long term recommendation is to use an nfs mount or something that supports RWM
    # import_path = "0/alma-base-image-0.0.16.qcow2"
  }

  network_interface {
    mac_address = "1a:2b:3c:4e:5f:61"
    bridge = "shared"
    firewall = true
    order = 0
    mtu = 1412
  }
}
