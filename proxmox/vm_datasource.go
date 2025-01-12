package proxmox

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-proxmox/proxmox_client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &qemuDataSource{}
	_ datasource.DataSourceWithConfigure = &qemuDataSource{}
)

type qemuDataSource struct {
	client *proxmox_client.Client
}

func NewVMDataSource() datasource.DataSource {
	return &qemuDataSource{}
}

func (d *qemuDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_vm"
}

func (d *qemuDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var plan *VmModel
	diags := request.Config.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	if plan == nil {
		response.Diagnostics.AddError("no datasource data passed in", "vm name is nil")
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Node name is %s", plan.NodeName.ValueString()))

	if !plan.NodeName.IsNull() {
		vmResponse, searchVmError := d.client.GetVmById(plan.NodeName.ValueString(), plan.VmId.ValueString())

		if searchVmError != nil {
			response.Diagnostics.AddError(fmt.Sprintf("Error retrieving vms from node %s", plan.NodeName.ValueString()), searchVmError.Error())
			return
		}

		updateVmModelFromResponse(plan, *vmResponse, &ctx)
	} else {
		nodeList, listNodesError := d.client.ListNodes()

		if listNodesError != nil {
			response.Diagnostics.AddError("Failed to list nodes in proxmox cluster", listNodesError.Error())
			return
		}

		found := false
		for _, node := range nodeList.Data {

			tflog.Debug(ctx, fmt.Sprintf("Node name is %s", node.Node))

			vmResponse, searchVmError := d.client.GetVmById(node.Node, plan.VmId.ValueString())

			if searchVmError != nil && !strings.Contains(searchVmError.Error(), fmt.Sprintf("500 Configuration file 'nodes/%s/qemu-server/%s.conf' does not exist", node.Node, plan.VmId.ValueString())) {
				response.Diagnostics.AddError("Failed to search for node that VM lives on", searchVmError.Error())
				return
			}
			if searchVmError == nil {
				found = true
				plan.NodeName = types.StringValue(node.Node)
				updateVmModelFromResponse(plan, *vmResponse, &ctx)
				break
			}

		}

		if !found {
			response.Diagnostics.AddError(fmt.Sprintf("Could not find vm for id %s within the cluster", plan.VmId.ValueString()), "not found")
			return
		}
	}

	diags = response.State.Set(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *qemuDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*proxmox_client.Client)
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
			"disk": schema.ListNestedBlock{
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
			"auto_start": schema.BoolAttribute{
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
		},
	}
}
