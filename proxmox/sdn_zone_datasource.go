package proxmox

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-proxmox/proxmox_client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &sdnZoneDatasource{}
	_ datasource.DataSourceWithConfigure = &sdnZoneDatasource{}
)

type sdnZoneDatasource struct {
	client *proxmox_client.Client
}

func NewSdnZoneDatasource() datasource.DataSource {
	return &sdnZoneDatasource{}
}

func (d *sdnZoneDatasource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_sdn_zone"
}

func (d *sdnZoneDatasource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var plan *sdnZone
	diags := request.Config.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	if plan == nil {
		response.Diagnostics.AddError("no datasource data passed in", "sdn zone is nil")
		return
	}

	zoneResponse, getZoneError := d.client.GetSdnZone(plan.Zone.ValueString())

	if getZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve zone %s", plan.Zone.ValueString()), getZoneError.Error())
		return
	}

	updateSdnZoneFromResponse(plan, ctx, *zoneResponse)

	diags = response.State.Set(ctx, plan)
	response.Diagnostics.Append(diags...)
}

func (d *sdnZoneDatasource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*proxmox_client.Client)
}

func (d *sdnZoneDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{Computed: true},
			"zone": schema.StringAttribute{Required: true},
			"ipam": schema.StringAttribute{Computed: true},
			"nodes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"peers": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
