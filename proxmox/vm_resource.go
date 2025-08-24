package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-proxmox/proxmox_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
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
	client *proxmox_client.Client
}

// Configure adds the provider configured client to the resource.
func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*proxmox_client.Client)
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
							PlanModifiers:       []planmodifier.Int64{},
							Default:             nil,
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

	var plan VmModel
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	//TODO: detect need for cloudinit disk
	qemuVmCreationRequest := createVmRequest(plan, &ctx, true, true)

	upid, vmCreationError := r.client.CreateVm(qemuVmCreationRequest, plan.NodeName.ValueString())

	if vmCreationError != nil {
		response.Diagnostics.AddError("Failed to create proxmox vm, error response received", vmCreationError.Error())
		return
	}

	taskCompletionError := waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

	if taskCompletionError != nil {
		response.Diagnostics.AddError("Creation of requested VM failed", taskCompletionError.Error())
		return
	}

	for _, disk := range plan.Disks {
		tflog.Info(ctx, fmt.Sprintf("Import from is %s", disk.ImportFrom.ValueString()))
		if disk.ImportFrom.ValueString() != "" {
			params := url.Values{}
			tflog.Info(ctx, "Resizing imported disk")

			params.Add("disk", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
			params.Add("size", disk.Size.ValueString())

			taskResponse, resizeDiskError := r.client.ResizeVmDisk(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

			if resizeDiskError != nil {
				response.Diagnostics.AddError("Failed to resize imported disk", resizeDiskError.Error())
				return
			}

			taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *taskResponse, r.client)

			if taskCompletionError != nil {
				response.Diagnostics.AddError("Failed to wait for resize task completion", taskCompletionError.Error())
				return
			}
		}
	}

	vmResponse, searchVmError := r.client.GetVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if searchVmError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Error retrieving vms from node %s with id %s", plan.NodeName.ValueString(), plan.VmId.ValueString()), searchVmError.Error())
		return
	}
	plan = assignDiskIds(plan)
	plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)
	diags = response.State.Set(ctx, &plan) //checkpoint
	vmStatus, getStatusError := r.client.GetVmStatus(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if getStatusError != nil {
		diags.AddError("Failed to retrieve vm statius", getStatusError.Error())
		return
	}

	if plan.PowerState.ValueString() == "running" && vmStatus != "running" {
		startVmError := startVm(plan.NodeName.ValueString(), plan.VmId.ValueString(), response.Diagnostics, r.client)
		if startVmError != nil {
			return
		}
	}

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *vmResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {

	var plan VmModel

	diags := request.State.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if !plan.NodeName.IsNull() {
		vmResponse, searchVmError := r.client.GetVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

		if searchVmError != nil {
			response.Diagnostics.AddError(fmt.Sprintf("Error retrieving vms from node %s", plan.NodeName.ValueString()), searchVmError.Error())
			return
		}

		plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)

	} else {
		nodeList, listNodesError := r.client.ListNodes()

		if listNodesError != nil {
			response.Diagnostics.AddError("Failed to list nodes in proxmox cluster", listNodesError.Error())
			return
		}

		found := false
		for _, node := range nodeList.Data {

			tflog.Debug(ctx, fmt.Sprintf("Node name is %s", node.Node))

			vmResponse, searchVmError := r.client.GetVmById(node.Node, plan.VmId.ValueString())

			if searchVmError != nil && !strings.Contains(searchVmError.Error(), fmt.Sprintf("500 Configuration file 'nodes/%s/qemu-server/%s.conf' does not exist", node.Node, plan.VmId.ValueString())) {
				response.Diagnostics.AddError("Failed to search for node that VM lives on", searchVmError.Error())
				return
			}
			if searchVmError == nil {
				found = true
				plan.NodeName = types.StringValue(node.Node)
				plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)
				break
			}

		}

		if !found {
			response.Diagnostics.AddError(fmt.Sprintf("Could not find vm for id %s within the cluster", plan.VmId.ValueString()), "not found")
			return
		}
	}

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *vmResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state VmModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	vmResponse, searchVmError := r.client.GetVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if searchVmError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Error retrieving vms from node %s with id %s", plan.NodeName.ValueString(), plan.VmId.ValueString()), searchVmError.Error())
		return
	}

	qemuVmCreationRequest := createVmRequest(plan, &ctx, false, false)

	var disks = updateDisksFromQemuResponse(vmResponse.Data.OtherFields, plan)

	diskChangeMapping, disksToBeRemoved := mapPlannedDisksToExisting(plan.Disks, disks)

	var disksToAdd []VmDisk
	var disksToUpdate []VmDisk
	var disksToResize []VmDisk
	bootDisks := make([]types.String, 0, len(plan.BootOrder.Elements()))
	_ = plan.BootOrder.ElementsAs(ctx, &bootDisks, false)

	for plannedDiskIndex, existingDiskIndex := range diskChangeMapping {
		plannedDisk := plan.Disks[plannedDiskIndex]
		if existingDiskIndex == -1 {
			disksToAdd = append(disksToAdd, plannedDisk)
			continue
		}

		existingDisk := disks[existingDiskIndex]
		stateDisk := getDiskFromState(state, getDiskName(plannedDisk))

		if stateDisk.Size.ValueString() == "" {
			tflog.Info(ctx, fmt.Sprintf("disk %s not found in the statefile skipping", getDiskName(plannedDisk)))
			tflog.Info(ctx, fmt.Sprintf("existing disk is %s", getDiskName(existingDisk)))
			disksToAdd = append(disksToAdd, plannedDisk)
			continue
		}

		tflog.Info(ctx, fmt.Sprintf("Disk %s needs update %v", getDiskName(plannedDisk), diskNeedsUpdate(plannedDisk, existingDisk)))
		isImported := plannedDisk.ImportFrom.ValueString() != "" && stateDisk.ImportFrom.ValueString() != ""
		importLocationChanged := plannedDisk.ImportFrom.ValueString() != stateDisk.ImportFrom.ValueString()
		importPathChanged := plannedDisk.Path.ValueString() != stateDisk.Path.ValueString()
		tflog.Info(ctx, fmt.Sprintf("Import location or path has changes %v", isImported && (importLocationChanged || importPathChanged)))
		if diskNeedsUpdate(plannedDisk, existingDisk) || (isImported && (importLocationChanged || importPathChanged)) {

			diskName := getDiskName(plannedDisk)
			tflog.Info(ctx, fmt.Sprintf("Planned disk %s is imported from %s, existing disk is imported from %s", diskName, plannedDisk.ImportFrom.ValueString(), stateDisk.ImportFrom.ValueString()))
			tflog.Info(ctx, fmt.Sprintf("Planned disk %s is import path is %s, existing disk is import path is %s", diskName, plannedDisk.Path.ValueString(), stateDisk.Path.ValueString()))
			tflog.Info(ctx, fmt.Sprintf("Disk %s is imported %v", diskName, plannedDisk.ImportFrom.ValueString() != "" && stateDisk.ImportFrom.ValueString() != ""))

			if isImported && (importLocationChanged || importPathChanged) {
				tflog.Info(ctx, fmt.Sprintf("Disk %s will be removed and reimported", getDiskName(plannedDisk)))
				disksToBeRemoved = append(disksToBeRemoved, existingDisk)
				disksToAdd = append(disksToAdd, plannedDisk)
				continue
			}
			plannedSize := convertSizeToGibibytes(plannedDisk.Size.ValueString())
			existingSize := convertSizeToGibibytes(existingDisk.Size.ValueString())
			if plannedSize < existingSize {
				diags.AddError("Cannot reduce the size of an existing volume", fmt.Sprintf("%d < %d", convertSizeToGibibytes(plannedDisk.Size.ValueString()), convertSizeToGibibytes(existingDisk.Size.ValueString())))
				return
			}
			if plannedSize > existingSize {
				disksToResize = append(disksToResize, plannedDisk)
			}

			disksToUpdate = append(disksToUpdate, plannedDisk)
		}
	}

	var upid *string
	var vmCreationError, taskCompletionError error

	if len(state.Disks) != len(plan.Disks) {
		for _, stateDisk := range state.Disks {
			stateIndex := findDiskIndex(plan.Disks, stateDisk)
			if stateIndex == -1 {
				disksToBeRemoved = append(disksToBeRemoved, stateDisk)
			}
		}
	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to remove", len(disksToBeRemoved)))
	vmShutdown := len(disksToBeRemoved) > 0 || len(disksToUpdate) > 0 || len(disksToAdd) > 0 || len(disksToResize) > 0
	if vmShutdown || state.PowerState.ValueString() == "running" && plan.PowerState.ValueString() == "stopped" {
		shutdownError := shutdownVm(plan.NodeName.ValueString(), plan.VmId.ValueString(), diags, r.client)
		if shutdownError != nil {
			return
		}
	}

	for _, disk := range disksToBeRemoved {
		tflog.Info(ctx, fmt.Sprintf("Detaching disk %s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
		params := url.Values{}
		params.Add("delete", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))

		upid, vmCreationError = r.client.UpdateVm(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

		if vmCreationError != nil {
			diags.AddError(fmt.Sprintf("Failed to update proxmox vm %s%d, error response received", disk.BusType.ValueString(), disk.Order.ValueInt64()), vmCreationError.Error())
			return
		}

		tflog.Info(ctx, fmt.Sprintf("Detaching disk %s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
		params = url.Values{}
		params.Add("delete", "unused0")

		upid, vmCreationError = r.client.UpdateVm(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

		if vmCreationError != nil {
			diags.AddError(fmt.Sprintf("Failed to update proxmox vm %s%d, error response received", disk.BusType.ValueString(), disk.Order.ValueInt64()), vmCreationError.Error())
			return
		}

		taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

		if taskCompletionError != nil {
			response.Diagnostics.AddError("Creation of requested VM disk failed", taskCompletionError.Error())
			return
		}

	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to add", len(disksToAdd)))

	for _, disk := range disksToAdd {
		params := url.Values{}

		attachVmDiskRequests([]VmDisk{disk}, &params, plan.VmId.ValueString(), false, true)

		tflog.Debug(ctx, fmt.Sprintf("Updating disk %s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))

		upid, vmCreationError = r.client.UpdateVm(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

		if vmCreationError != nil {
			diags.AddError(fmt.Sprintf("Failed to update proxmox vm %s%d, error response received", disk.BusType.ValueString(), disk.Order.ValueInt64()), vmCreationError.Error())
			return
		}

		taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

		if taskCompletionError != nil {
			diags.AddError("Creation of requested VM disk failed", taskCompletionError.Error())
			return
		}

		if disk.ImportFrom.ValueString() != "" {
			params := url.Values{}
			tflog.Info(ctx, "Resizing imported disk")

			params.Add("disk", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
			params.Add("size", disk.Size.ValueString())

			taskResponse, resizeDiskError := r.client.ResizeVmDisk(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

			if resizeDiskError != nil {
				response.Diagnostics.AddError("Failed to resize imported disk", resizeDiskError.Error())
				return
			}

			taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *taskResponse, r.client)

			if taskCompletionError != nil {
				response.Diagnostics.AddError("Failed to wait for resize task completion", taskCompletionError.Error())
				return
			}
		}

	}

	tflog.Info(ctx, fmt.Sprintf("There are %d disks to update", len(disksToUpdate)))

	for _, disk := range disksToUpdate {
		params := url.Values{}

		attachVmDiskRequests([]VmDisk{disk}, &params, plan.VmId.ValueString(), false, false)

		upid, vmCreationError = r.client.UpdateVm(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

		if vmCreationError != nil {
			diags.AddError(fmt.Sprintf("Failed to update proxmox vm %s%d, error response received", disk.BusType.ValueString(), disk.Order.ValueInt64()), vmCreationError.Error())
			return
		}

		taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

		if taskCompletionError != nil {
			response.Diagnostics.AddError("Creation of requested VM disk failed", taskCompletionError.Error())
			return
		}

	}

	for _, disk := range disksToResize {
		params := url.Values{}
		params.Add("size", disk.Size.ValueString())
		params.Add("disk", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))

		upid, vmCreationError = r.client.ResizeVmDisk(params, plan.NodeName.ValueString(), plan.VmId.ValueString())

		if vmCreationError != nil {
			diags.AddError(fmt.Sprintf("Failed to update proxmox vm %s%d, error response received", disk.BusType.ValueString(), disk.Order.ValueInt64()), vmCreationError.Error())
			return
		}

		taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

		if taskCompletionError != nil {
			response.Diagnostics.AddError("Creation of requested VM disk failed", taskCompletionError.Error())
			return
		}

	}
	upid, vmCreationError = r.client.UpdateVm(qemuVmCreationRequest, plan.NodeName.ValueString(), plan.VmId.ValueString())

	if vmCreationError != nil {
		diags.AddError("Failed to update proxmox vm, error response received", vmCreationError.Error())
		return
	}

	taskCompletionError = waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

	if taskCompletionError != nil {
		diags.AddError("Creation of requested VM failed", taskCompletionError.Error())
		return
	}

	if vmShutdown || plan.PowerState.ValueString() == "running" || state.PowerState.ValueString() == "stopped" && plan.PowerState.ValueString() == "running" {
		startError := startVm(plan.NodeName.ValueString(), plan.VmId.ValueString(), diags, r.client)
		if startError != nil {
			return
		}
		plan.PowerState = types.StringValue("running")
	}

	vmResponse, searchVmError = r.client.GetVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if searchVmError != nil {
		diags.AddError(fmt.Sprintf("Error retrieving vms from node %s with id %s", plan.NodeName.ValueString(), plan.VmId.ValueString()), searchVmError.Error())
		return
	}
	plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)
	sort.Slice(plan.Disks, func(i int, j int) bool {
		return plan.Disks[i].Order.ValueInt64() < plan.Disks[j].Order.ValueInt64()
	})
	for i, disk := range plan.Disks {
		index := findDiskIndex(state.Disks, disk)
		if index != -1 && (state.Disks[index].Path.ValueString() != "" || state.Disks[index].ImportFrom.ValueString() != "") {
			plan.Disks[i].Path = state.Disks[index].Path
			plan.Disks[i].ImportFrom = state.Disks[index].ImportFrom
		}
	}
	plan.Disks = disks
	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *vmResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {

	var plan VmModel
	diags := request.State.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	vmStatus, getStatusError := r.client.GetVmStatus(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if getStatusError != nil {
		diags.AddError("Failed to retrieve VM status prior to deletion", getStatusError.Error())
		return
	}

	if vmStatus != "stopped" {
		shutdownError := shutdownVm(plan.NodeName.ValueString(), plan.VmId.ValueString(), diags, r.client)
		if shutdownError != nil {
			return
		}
	}

	upid, vmDeletionError := r.client.DeleteVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

	if vmDeletionError != nil {
		response.Diagnostics.AddError(fmt.Sprintf("Failed to delete proxmox vm %s, error response received", plan.VmId.ValueString()), vmDeletionError.Error())
		return
	}

	taskCompletionError := waitForTaskCompletion(plan.NodeName.ValueString(), *upid, r.client)

	if taskCompletionError != nil {
		response.Diagnostics.AddError("Creation of requested VM disk failed", taskCompletionError.Error())
		return
	}

	response.Diagnostics.Append(diags...)
	if diags.HasError() {

		return
	}
	response.State.RemoveResource(ctx)
}

func (r *vmResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	var plan VmModel

	vmId, intParseError := strconv.ParseInt(request.ID, 10, 64)

	if intParseError != nil {
		response.Diagnostics.AddError("Unable to import vm, vm id parsing failed", intParseError.Error())
		return
	}

	plan.VmId = types.StringValue(request.ID)

	nodeList, listNodesError := r.client.ListNodes()

	if listNodesError != nil {
		response.Diagnostics.AddError("Failed to list nodes in proxmox cluster", listNodesError.Error())
		return
	}

	found := false
	for _, node := range nodeList.Data {

		tflog.Debug(ctx, fmt.Sprintf("Node name is %s", node.Node))

		vmResponse, searchVmError := r.client.GetVmById(node.Node, plan.VmId.ValueString())

		if searchVmError != nil && !strings.Contains(searchVmError.Error(), fmt.Sprintf("500 Configuration file 'nodes/%s/qemu-server/%d.conf' does not exist", node.Node, vmId)) {
			response.Diagnostics.AddError("Failed to search for node that VM lives on", searchVmError.Error())
			return
		}
		if searchVmError == nil {
			found = true
			plan.NodeName = types.StringValue(node.Node)
			plan = updateVmModelFromResponse(plan, *vmResponse, &ctx)
			break
		}

	}

	if !found {
		response.Diagnostics.AddError(fmt.Sprintf("Could not find vm for id %d within the cluster", vmId), "not found")
		return
	}

	response.State.Set(ctx, plan)
	resource.ImportStatePassthroughID(ctx, path.Root("vm_id"), request, response)
}
