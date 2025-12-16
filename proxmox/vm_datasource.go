package proxmox

import (
	"context"
	"fmt"
	"terraform-provider-proxmox/proxmox_client"
	"terraform-provider-proxmox/services"
	"terraform-provider-proxmox/services/vm"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &qemuDataSource{}
	_ datasource.DataSourceWithConfigure = &qemuDataSource{}
)

type qemuDataSource struct {
	vmService     vm.VmService
	vmDiskService vm.DiskService
}

func NewVMDataSource() datasource.DataSource {
	return &qemuDataSource{}
}

func (d *qemuDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_vm"
}

func (d *qemuDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var plan proxmoxTypes.VmModel
	var currentState proxmoxTypes.VmModel = proxmoxTypes.VmModel{}
	diags := request.Config.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	tflog.Debug(ctx, fmt.Sprintf("Node name is %s", plan.NodeName.ValueString()))

	qemuResponse, nodeName, getVmError := d.vmService.GetVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())

	if getVmError != nil {
		response.Diagnostics.AddError("Failed to find requested vm", getVmError.Error())
		return
	}
	currentState.VmId = plan.VmId
	currentState.NodeName = types.StringValue(*nodeName)

	d.vmService.UpdateVmModelFromResponse(&currentState, &plan, qemuResponse)
	updatePowerStateError := d.vmService.UpdatePowerState(&currentState)

	if updatePowerStateError != nil {
		response.Diagnostics.AddError("Failed to update VM power state.", updatePowerStateError.Error())
	}

	diags = response.State.Set(ctx, &currentState)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *qemuDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	proxmoxClient := req.ProviderData.(proxmox_client.ProxmoxClient)
	proxmoxUtils := services.NewProxmoxUtilService()
	taskService := services.NewTaskService(proxmoxClient)
	diskService := vm.NewDiskService(ctx, proxmoxClient, proxmoxUtils, taskService)
	d.vmService = vm.NewVmService(ctx, proxmoxClient, diskService, proxmoxUtils, taskService)
}

// Schema defines the schema for the data source.
func (d *qemuDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"ip_config": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Computed: true,
						},
						"gateway": schema.StringAttribute{
							Computed: true,
						},
						"order": schema.Int64Attribute{
							Computed: true,
						},
					},
				},
			},
			"disk": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed: true,
						},
						"bus_type": schema.StringAttribute{
							Computed: true,
						},
						"storage_location": schema.StringAttribute{
							Computed: true,
						},
						"io_thread": schema.BoolAttribute{
							Computed: true,
						},
						"size": schema.StringAttribute{
							Computed: true,
						},
						"cache": schema.StringAttribute{
							Computed: true,
						},
						"async_io": schema.StringAttribute{
							Computed: true,
						},
						"replicate": schema.BoolAttribute{
							Computed: true,
						},
						"read_only": schema.BoolAttribute{
							Computed: true,
						},
						"ssd_emulation": schema.BoolAttribute{
							Computed: true,
						},
						"backup_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"discard_enabled": schema.BoolAttribute{
							Computed: true,
						},
						"order":       schema.Int64Attribute{Computed: true},
						"import_from": schema.StringAttribute{Computed: true},
						"import_path": schema.StringAttribute{Computed: true},
					},
				},
			},
			"network_interface": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"mac_address": schema.StringAttribute{
							Computed: true,
						},
						"bridge": schema.StringAttribute{
							Computed: true,
						},
						"firewall": schema.BoolAttribute{
							Computed: true,
						},
						"order": schema.Int64Attribute{Computed: true},
						"type":  schema.StringAttribute{Computed: true},
						"mtu":   schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Computed: true,
			},
			"vmgenid": schema.StringAttribute{
				Computed: true,
			},
			"cores": schema.Int64Attribute{
				Computed: true,
			},
			"memory": schema.Int64Attribute{
				Computed: true,
			},
			"os_type": schema.StringAttribute{
				Computed: true,
			},
			"description": schema.StringAttribute{
				Computed: true,
			},
			"node_name": schema.StringAttribute{
				Required: true,
			},
			"vm_id": schema.StringAttribute{
				Required: true,
			},
			"start_on_boot": schema.BoolAttribute{
				Computed: true,
			},
			"numa_active": schema.BoolAttribute{
				Computed: true,
			},
			"cpu_type": schema.StringAttribute{
				Computed: true,
			},
			"sockets": schema.Int64Attribute{
				Computed: true,
			},
			"scsi_hw": schema.StringAttribute{
				Computed: true,
			},
			"qemu_agent_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"tags": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"boot_order": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"nameserver": schema.StringAttribute{Computed: true},
			"perform_cloud_init_upgrade": schema.BoolAttribute{
				Computed: true,
			},
			"kvm": schema.BoolAttribute{
				Computed: true,
			},
			"acpi": schema.BoolAttribute{
				Computed: true,
			},
			"cpu_limit": schema.Int64Attribute{
				Computed: true,
			},
			"bios": schema.StringAttribute{
				Computed: true,
			},
			"host_startup_order": schema.Int64Attribute{
				Computed: true,
			},
			"ssh_keys": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"protection": schema.BoolAttribute{
				Computed: true,
			},
			"default_user": schema.StringAttribute{Computed: true},
			"cloud_init_storage_name": schema.StringAttribute{
				Computed: true,
			},
			"power_state": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}
