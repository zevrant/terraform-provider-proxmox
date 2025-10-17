package proxmox

import (
	"context"
	"terraform-provider-proxmox/proxmox_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &proxmoxProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &proxmoxProvider{}
}

// proxmoxProvider is the provider implementation.
type proxmoxProvider struct{}

// DataSources defines the data sources implemented in the provider.
func (p *proxmoxProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVMDataSource,
		NewSdnZoneDatasource,
		NewNodeDataSource,
		NewHealthCheckSystemdDatasource,
	}
}

func (p *proxmoxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVmResource,
		NewSdnZoneResource,
	}
}

// proxmoxProviderModel maps provider schema data to a Go type.
type proxmoxProviderModel struct {
	Host      types.String `tfsdk:"host"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	VerifyTLS types.Bool   `tfsdk:"verify_tls"`
}

// Metadata returns the provider type name.
func (p *proxmoxProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "proxmox"
}

// Schema defines the provider-level schema for configuration data.
func (p *proxmoxProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"verify_tls": schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (p *proxmoxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config proxmoxProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := config.Host.ValueString()
	username := config.Username.ValueString()
	password := config.Password.ValueString()
	verifyTls := config.VerifyTLS.ValueBool()

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing proxmox API Host",
			"The provider cannot create the proxmox API client as there is a missing or empty value for the proxmox API host. "+
				"Set the host value in the configuration or use the PROXMOX_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing proxmox API Username",
			"The provider cannot create the proxmox API client as there is a missing or empty value for the proxmox API username. "+
				"Set the username value in the configuration or use the PROXMOX_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing proxmox API Password",
			"The provider cannot create the proxmox API client as there is a missing or empty value for the proxmox API password. "+
				"Set the password value in the configuration or use the PROXMOX_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a new proxmox client using the configuration values
	client := proxmox_client.NewClient(&host, &username, &password, &verifyTls, ctx)

	// Make the proxmox client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}
