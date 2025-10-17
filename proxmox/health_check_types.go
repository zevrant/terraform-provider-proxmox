package proxmox

import "github.com/hashicorp/terraform-plugin-framework/types"

type HealthCheckSystemd struct {
	Address     types.String `tfsdk:"address"`
	Path        types.String `tfsdk:"path"`
	TlsEnabled  types.Bool   `tfsdk:"tls_enabled"`
	ServiceName types.String `tfsdk:"service_name"`
	CustomPort  types.Int64  `tfsdk:"custom_port"`
}
