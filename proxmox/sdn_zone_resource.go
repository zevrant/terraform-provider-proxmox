package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"terraform-provider-proxmox/proxmox_client"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &sdnZoneResource{}
	_ resource.ResourceWithConfigure   = &sdnZoneResource{}
	_ resource.ResourceWithImportState = &sdnZoneResource{}
)

func NewSdnZoneResource() resource.Resource {
	return &sdnZoneResource{}
}

// sdnZoneResource is the resource implementation.
type sdnZoneResource struct {
	client proxmox_client.ProxmoxClient
}

// Configure adds the provider configured client to the resource.
func (r *sdnZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(proxmox_client.ProxmoxClient)
}

// Metadata returns the resource type name.
func (r *sdnZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = req.ProviderTypeName + "_sdn_zone"
}

// Schema defines the schema for the resource.
func (r *sdnZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{Required: true},
			"zone": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ipam": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("pve"),
			},
			"nodes": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			"peers": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *sdnZoneResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {

	var plan sdnZone
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	params := url.Values{}

	assembleCreateSdnZoneRequest(&params, plan, ctx)

	createZoneError := r.client.CreateSdnZone(params)

	if createZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to create SDN %s Zone", plan.Type.ValueString()), createZoneError.Error())
	}

	zoneResponse, getZoneError := r.client.GetSdnZone(plan.Zone.ValueString())

	if getZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve zone %s", plan.Zone.ValueString()), getZoneError.Error())
		return
	}

	updateSdnZoneFromResponse(&plan, ctx, *zoneResponse)

	diags = response.State.Set(ctx, plan)
	response.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *sdnZoneResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {

	var plan sdnZone

	diags := request.State.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneResponse, getZoneError := r.client.GetSdnZone(plan.Zone.ValueString())

	if getZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve zone %s", plan.Zone.ValueString()), getZoneError.Error())
		return
	}

	updateSdnZoneFromResponse(&plan, ctx, *zoneResponse)

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *sdnZoneResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan *sdnZone
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	params := url.Values{}

	assembleCreateSdnZoneRequest(&params, *plan, ctx)

	params.Del("type")

	createZoneError := r.client.UpdateSdnZone(params)

	if createZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to update SDN %s Zone", plan.Type.ValueString()), createZoneError.Error())
	}

	zoneResponse, getZoneError := r.client.GetSdnZone(plan.Zone.ValueString())

	if getZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve zone %s", plan.Zone.ValueString()), getZoneError.Error())
		return
	}

	updateSdnZoneFromResponse(plan, ctx, *zoneResponse)

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *sdnZoneResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {

	var plan sdnZone
	diags := request.State.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	deleteZoneError := r.client.DeleteSdnZone(plan.Zone.ValueString())

	if deleteZoneError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to delete SDN Zone %s", plan.Zone.ValueString()), deleteZoneError.Error())
		return
	}

	response.Diagnostics.Append(diags...)
	if diags.HasError() {

		return
	}
	response.State.RemoveResource(ctx)
}

func (r *sdnZoneResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	var plan proxmoxTypes.VmModel

	response.State.Set(ctx, plan)
	resource.ImportStatePassthroughID(ctx, path.Root("vm_id"), request, response)
}
