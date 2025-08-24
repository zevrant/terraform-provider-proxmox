package main

import (
	"context"
	"os"
	"strings"
	"terraform-provider-proxmox/proxmox"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	debug_mode := "true" == strings.ToLower(os.Getenv("TERRAFORM_DEBUG_MODE"))
	providerserver.Serve(context.Background(), proxmox.New, providerserver.ServeOpts{
		Address: "app.terraform.io/zevrant-services/proxmox",
		Debug:   debug_mode,
	})

}
