package proxmox

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-proxmox/proxmox_client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &nodeDataSource{}
	_ datasource.DataSourceWithConfigure = &nodeDataSource{}
)

type nodeDataSource struct {
	client proxmox_client.ProxmoxClient
}

type NodeModel struct {
	Name           types.String  `tfsdk:"name"`
	Status         types.String  `tfsdk:"status"`
	Cpu            types.Float64 `tfsdk:"cpu"`
	Level          types.String  `tfsdk:"level"`
	MaxCpu         types.Int64   `tfsdk:"max_cpu"`
	MaxMemory      types.Int64   `tfsdk:"max_memory"`
	Memory         types.Int64   `tfsdk:"memory"`
	SslFingerprint types.String  `tfsdk:"ssl_fingerprint"`
	Uptime         types.Int64   `tfsdk:"uptime"`
	NetworkAddress types.String  `tfsdk:"network_address"`
}

func NewNodeDataSource() datasource.DataSource {
	return &nodeDataSource{}
}

func (d *nodeDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_node"
}

func (d *nodeDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var plan *NodeModel
	diags := request.Config.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)

	if plan == nil {
		response.Diagnostics.AddError("no datasource data passed in", "vm name is nil")
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Node name is %s", plan.Name.ValueString()))

	nodes, listNodesError := d.client.ListNodes()

	if listNodesError != nil {
		response.Diagnostics.AddError("Failed to list nodes", listNodesError.Error())
		return
	}

	found := false
	for _, node := range nodes.Data {
		if node.Node == plan.Name.ValueString() {
			found = true
			plan.Status = types.StringValue(node.Status)
			plan.Cpu = types.Float64Value(node.Cpu)
			plan.Memory = types.Int64Value(node.Mem)
			plan.Level = types.StringValue(node.Level)
			plan.MaxCpu = types.Int64Value(int64(node.MaxCpu))
			plan.MaxMemory = types.Int64Value(int64(node.MaxMem))
			plan.SslFingerprint = types.StringValue(node.SslFingerprint)
			plan.Uptime = types.Int64Value(node.Uptime)
		}
	}

	if !found {
		response.Diagnostics.AddError("Failed to retrieve node information", "not found")
		return
	}

	networkConfig, getNetworkConfigError := d.client.GetNodeNetworkConfig(plan.Name.ValueString())

	if getNetworkConfigError != nil {
		response.Diagnostics.AddError("Failed to retrieve networkconfig for node", getNetworkConfigError.Error())
		return
	}

	var enabledInterfaces []string

	for _, netConfig := range networkConfig.Data {
		if netConfig.Address != "" {
			enabledInterfaces = append(enabledInterfaces, netConfig.Address)
		}
	}

	plan.NetworkAddress = types.StringValue(enabledInterfaces[0])

	diags = response.State.Set(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

// Configure adds the provider configured client to the data source.
func (d *nodeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(proxmox_client.ProxmoxClient)
}

// Schema defines the schema for the data source.
func (d *nodeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"status":          schema.StringAttribute{Computed: true},
			"cpu":             schema.Float64Attribute{Computed: true},
			"level":           schema.StringAttribute{Computed: true},
			"max_cpu":         schema.Int64Attribute{Computed: true},
			"max_memory":      schema.Int64Attribute{Computed: true},
			"memory":          schema.Int64Attribute{Computed: true},
			"ssl_fingerprint": schema.StringAttribute{Computed: true},
			"uptime":          schema.Int64Attribute{Computed: true},
			"network_address": schema.StringAttribute{Computed: true},
		},
	}
}
