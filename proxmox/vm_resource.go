package proxmox

import (
	"context"
	"fmt"
	"terraform-provider-proxmox/proxmox_client"
	"terraform-provider-proxmox/services"
	"terraform-provider-proxmox/services/vm"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &vmResource{}
	_ resource.ResourceWithConfigure   = &vmResource{}
	_ resource.ResourceWithImportState = &vmResource{}
)

func NewVmResource() resource.Resource {
	return &vmResource{}
}

// vmResource is the resource implementation.
type vmResource struct {
	vmService   vm.VmService
	diskService vm.DiskService
}

// Configure adds the provider configured client to the resource.
func (r *vmResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	proxmoxClient := req.ProviderData.(proxmox_client.ProxmoxClient)
	proxmoxUtils := services.NewProxmoxUtilService()
	taskService := services.NewTaskService(proxmoxClient)
	r.diskService = vm.NewDiskService(ctx, proxmoxClient, proxmoxUtils, taskService)
	r.vmService = vm.NewVmService(ctx, proxmoxClient, r.diskService, proxmoxUtils, taskService)
}

// Metadata returns the resource type name.
func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = req.ProviderTypeName + "_vm"
}

// Schema defines the schema for the resource.
func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"ip_config": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Required: true,
						},
						"gateway": schema.StringAttribute{
							Required: true,
						},
						"order": schema.Int64Attribute{
							Required: true,
						},
					},
				},
			},
			"disk": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							CustomType:          nil,
							Required:            false,
							Optional:            false,
							Computed:            true,
							Sensitive:           false,
							Description:         "",
							MarkdownDescription: "",
							DeprecationMessage:  "",
							Validators:          nil,
							PlanModifiers:       []planmodifier.Int64{},
							Default:             nil,
							WriteOnly:           false,
						},
						"bus_type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("scsi"),
						},
						"storage_location": schema.StringAttribute{
							Required: true,
						},
						"io_thread": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(true),
						},
						"size": schema.StringAttribute{
							Required: true,
						},
						"cache": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("default"),
						},
						"async_io": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("default"),
						},
						"replicate": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(true),
						},
						"read_only": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
						"ssd_emulation": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
						"backup_enabled": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(true),
						},
						"discard_enabled": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
						"order": schema.Int64Attribute{Required: true},
						"import_from": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"import_path": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"network_interface": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"mac_address": schema.StringAttribute{
							Required: true,
						},
						"bridge": schema.StringAttribute{
							Required: true,
						},
						"firewall": schema.BoolAttribute{
							Optional: true,
							Computed: true,
							Default:  booldefault.StaticBool(true),
						},
						"order": schema.Int64Attribute{Required: true},
						"type": schema.StringAttribute{
							Computed: true,
							Optional: true,
							Default:  stringdefault.StaticString("virtio"),
						},
						"mtu": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"vmgenid": schema.StringAttribute{
				Computed: true,
			},
			"cores": schema.Int64Attribute{
				Required: true,
			},
			"memory": schema.Int64Attribute{
				Required: true,
			},
			"os_type": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"node_name": schema.StringAttribute{
				Required: true,
			},
			"vm_id": schema.StringAttribute{
				Required: true,
			},
			"start_on_boot": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"numa_active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"cpu_type": schema.StringAttribute{
				Required: true,
			},
			"sockets": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
			},
			"scsi_hw": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("virtio-scsi-single"),
			},
			"qemu_agent_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"boot_order": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			"nameserver": schema.StringAttribute{Required: true},
			"perform_cloud_init_upgrade": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"kvm": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"acpi": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"cpu_limit": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"bios": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("seabios"),
			},
			"host_startup_order": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"ssh_keys": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					sshKeyListValidator{},
				},
			},
			"protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"default_user": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"cloud_init_storage_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("local-zfs"),
			},
			"power_state": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("stopped"),
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *vmResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {

	var plan proxmoxTypes.VmModel

	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var currentState = plan
	createVmError := r.vmService.CreateVm(&plan)

	if createVmError != nil {
		response.Diagnostics.AddError("Failed to create vm", createVmError.Error())
		return
	}

	resizeDisksError := r.diskService.ResizeImportedDisks(plan.VmId.ValueStringPointer(), plan.NodeName.ValueStringPointer(), plan.Disks)

	if resizeDisksError != nil {
		response.Diagnostics.AddError("Failed to resize imported disks", resizeDisksError.Error())
		return
	}

	qemuResponse, _, getVmStateError := r.vmService.GetVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())

	if getVmStateError != nil {
		response.Diagnostics.AddError("Failed to refresh vm state after creation", getVmStateError.Error())
		diags = response.State.Set(ctx, &plan)
		response.Diagnostics.Append(diags...)
		return
	}

	r.vmService.UpdateVmModelFromResponse(&currentState, &plan, qemuResponse)

	updatePowerStateError := r.vmService.UpdatePowerState(&currentState)

	if updatePowerStateError != nil {
		response.Diagnostics.AddError("Failed to refresh vm current power state after creation", updatePowerStateError.Error())
		diags = response.State.Set(ctx, &currentState)
		response.Diagnostics.Append(diags...)
		return
	}

	matchPowerStateError := r.vmService.MatchVmPowerState(&plan, &currentState)

	if matchPowerStateError != nil {
		response.Diagnostics.AddError("Failed to match requested power state after vm creation", matchPowerStateError.Error())
	}

	diags = response.State.Set(ctx, &currentState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *vmResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {

	var state proxmoxTypes.VmModel

	diags := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	tflog.Debug(ctx, fmt.Sprintf("Node name is %s", state.NodeName.ValueString()))

	var currentState = state

	qemuResponse, nodeName, getVmError := r.vmService.GetVm(state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if getVmError != nil {
		response.Diagnostics.AddError("Failed to find requested vm", getVmError.Error())
		return
	}
	currentState.VmId = state.VmId
	currentState.NodeName = types.StringValue(*nodeName)

	r.vmService.UpdateVmModelFromResponse(&currentState, &state, qemuResponse)
	updatePowerStateError := r.vmService.UpdatePowerState(&currentState)

	if updatePowerStateError != nil {
		response.Diagnostics.AddError("Failed to update VM power state.", updatePowerStateError.Error())
	}

	diags = response.State.Set(ctx, &currentState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *vmResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var current, plan, state proxmoxTypes.VmModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	current = plan

	qemuResponse, _, getVmError := r.vmService.GetVm(state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if getVmError != nil {
		response.Diagnostics.AddError("Failed to refresh vm info prior to update", getVmError.Error())
		return
	}

	r.vmService.UpdateVmModelFromResponse(&current, &plan, qemuResponse)

	updateVmError := r.vmService.UpdateVm(&plan, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if updateVmError != nil {
		response.Diagnostics.AddError("Failed to update VM", updateVmError.Error())
		return
	}

	diskChanges := r.diskService.CompareVmDisks(&state, &plan)

	var toBeAdded, toBeUpdated, toBeRemoved, toBeResized []proxmoxTypes.VmDisk
	toBeAdded = diskChanges.ToBeAdded
	toBeUpdated = diskChanges.ToBeUpdated
	toBeResized = diskChanges.ToBeResized
	toBeRemoved = diskChanges.ToBeRemove
	toBeMigrated := diskChanges.ToBeMigrated

	migrationCount := 0
	for key, _ := range toBeMigrated {
		key.StorageLocation.ValueStringPointer()
		migrationCount += 1
	}

	if len(toBeAdded)+len(toBeUpdated)+len(toBeRemoved)+len(toBeResized)+migrationCount > 0 {
		tflog.Info(ctx, "Shutting down VM in order to provision disk changes")
		shutdownError := r.vmService.ShutdownVm(state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())
		if shutdownError != nil {
			tflog.Error(ctx, "Cannot perform disk updates, shutdown failed to complete")
			response.Diagnostics.AddError("Failed to shutdown Vm", shutdownError.Error())
			response.Diagnostics.Append(response.State.Set(ctx, &current)...)
			return
		}
	}
	moveDiskError := r.diskService.MoveDiskStorage(toBeMigrated, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if moveDiskError != nil {
		response.Diagnostics.AddError("Failed to move vm disk", moveDiskError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to remove", len(toBeRemoved)))

	diskDeletionError := r.diskService.DeleteVmDisks(toBeRemoved, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if diskDeletionError != nil {
		response.Diagnostics.AddError("Failed to delete Vm disk", diskDeletionError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to add", len(toBeAdded)))

	addDisksError := r.diskService.AddVmDisks(toBeAdded, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if addDisksError != nil {
		response.Diagnostics.AddError("Failed to add Vm disks", addDisksError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to update", len(toBeUpdated)))

	updateDisksError := r.diskService.UpdateVmDisks(toBeUpdated, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if updateDisksError != nil {
		response.Diagnostics.AddError("Failed to update vm disk configs", updateDisksError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	resizeDisksError := r.diskService.ResizeVmDisks(toBeResized, state.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())

	if resizeDisksError != nil {
		response.Diagnostics.AddError("Failed to resize VM Disks", resizeDisksError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	if state.NodeName.ValueString() != plan.NodeName.ValueString() {
		migrationError := r.vmService.MigrateVm(state.NodeName.ValueStringPointer(), plan.NodeName.ValueStringPointer(), state.VmId.ValueStringPointer())
		if migrationError != nil {
			response.Diagnostics.AddError("Failed to migrate VM", migrationError.Error())
			response.Diagnostics.Append(response.State.Set(ctx, &current)...)
			return
		}
	}

	qemuResponse, _, getVmError = r.vmService.GetVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())

	r.vmService.UpdateVmModelFromResponse(&current, &plan, qemuResponse)

	updatePowerStateError := r.vmService.UpdatePowerState(&current)

	if updatePowerStateError != nil {
		response.Diagnostics.AddError("Failed to update VM power state", updatePowerStateError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	if current.PowerState.ValueString() == "stopped" && plan.PowerState.ValueString() == "running" {
		startVmError := r.vmService.StartVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())
		if startVmError != nil {
			response.Diagnostics.AddError("Failed to start VM", startVmError.Error())
			response.Diagnostics.Append(response.State.Set(ctx, &current)...)
			return
		}
	}

	updatePowerStateError = r.vmService.UpdatePowerState(&current)

	if updatePowerStateError != nil {
		response.Diagnostics.AddError("Failed to update VM power state", updatePowerStateError.Error())
		response.Diagnostics.Append(response.State.Set(ctx, &current)...)
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &current)...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vmResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {

	var plan proxmoxTypes.VmModel
	diags := request.State.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	deleteVmError := r.vmService.DeleteVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())

	if deleteVmError != nil {
		response.Diagnostics.AddError("Failed to delete vm", deleteVmError.Error())
		return
	}

	response.Diagnostics.Append(diags...)
	if diags.HasError() {

		return
	}
	response.State.RemoveResource(ctx)
}

func (r *vmResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	//	// Retrieve import ID and save to id attribute
	//	var plan VmModel
	//
	//	vmId, intParseError := strconv.ParseInt(request.ID, 10, 64)
	//
	//	if intParseError != nil {
	//		response.Diagnostics.AddError("Unable to import vm, vm id parsing failed", intParseError.Error())
	//		return
	//	}
	//
	//	plan.VmId = types.StringValue(request.ID)
	//
	//	nodeList, listNodesError := r.client.ListNodes()
	//
	//	if listNodesError != nil {
	//		response.Diagnostics.AddError("Failed to list nodes in proxmox cluster", listNodesError.Error())
	//		return
	//	}
	//
	//	found := false
	//	for _, node := range nodeList.Data {
	//
	//		tflog.Debug(ctx, fmt.Sprintf("Node name is %s", node.Node))
	//
	//		vmResponse, searchVmError := r.client.GetVmById(node.Node, plan.VmId.ValueString())
	//
	//		if searchVmError != nil && !strings.Contains(searchVmError.Error(), fmt.Sprintf("500 Configuration file 'nodes/%s/qemu-server/%d.conf' does not exist", node.Node, vmId)) {
	//			response.Diagnostics.AddError("Failed to search for node that VM lives on", searchVmError.Error())
	//			return
	//		}
	//		if searchVmError == nil {
	//			found = true
	//			plan.NodeName = types.StringValue(node.Node)
	//			plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)
	//			break
	//		}
	//
	//	}
	//
	//	if !found {
	//		response.Diagnostics.AddError(fmt.Sprintf("Could not find vm for id %d within the cluster", vmId), "not found")
	//		return
	//	}
	//
	//	response.State.Set(ctx, plan)
	//	resource.ImportStatePassthroughID(ctx, path.Root("vm_id"), request, response)
}
