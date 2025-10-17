data proxmox_health_check_systemd named {
  service_name = "zs-vm-agent"
  address = "10.1.0.100"
  custom_port = 9100
  path = "/metrics"
}