package main

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"terraform-provider-proxmox/proxmox"
)

func main() {
	providerserver.Serve(context.Background(), proxmox.New, providerserver.ServeOpts{
		Address: "app.terraform.io/zevrant-services/proxmox",
		Debug:   true,
	})
}
