package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"os"
	"terraform-provider-proxmox/proxmox"
)

func main() {
	providerserver.Serve(context.Background(), proxmox.New, providerserver.ServeOpts{
		Address: fmt.Sprintf("app.terraform.io/zevrant-services%s/proxmox", os.Getenv("DEV_SUFFIX")),
	})
}
