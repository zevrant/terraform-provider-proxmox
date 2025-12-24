package proxmox

import (
	"context"
	"fmt"
	"terraform-provider-proxmox/proxmox_client"
	"terraform-provider-proxmox/services"
	proxmox_types "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &sdnZoneDatasource{}
	_ datasource.DataSourceWithConfigure = &sdnZoneDatasource{}
)

type qemuImageDatasource struct {
	storageService services.NodeStorageService
}

func NewQemuImage() datasource.DataSource {
	return &qemuImageDatasource{}
}

func (d *qemuImageDatasource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_qemu_image"
}

func (d *qemuImageDatasource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var plan *proxmox_types.QemuImage
	diags := request.Config.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	if plan == nil {
		response.Diagnostics.AddError("no datasource data passed in", "image is nil")
		return
	}

	images, listImagesError := d.storageService.ListImages(plan.StorageName.ValueStringPointer(), plan.Name.ValueStringPointer())

	if listImagesError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to list of images for name %s in storage %s", plan.Name.ValueString(), plan.StorageName.ValueString()), listImagesError.Error())
		return
	}

	latestImage := d.storageService.GetLatestImageFromImages(images)

	diags = response.State.Set(ctx, latestImage)
	response.Diagnostics.Append(diags...)
}

func (d *qemuImageDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := req.ProviderData.(proxmox_client.ProxmoxClient)
	d.storageService = services.NewNodeStorageService(client, ctx)
}

func (d *qemuImageDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"version":      schema.StringAttribute{Computed: true},
			"name":         schema.StringAttribute{Required: true},
			"storage_name": schema.StringAttribute{Required: true},
		},
	}
}
