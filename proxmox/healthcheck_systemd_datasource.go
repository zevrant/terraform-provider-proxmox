package proxmox

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-proxmox/healthcheck_client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &healthCheckSystemdDatasource{}
	_ datasource.DataSourceWithConfigure = &healthCheckSystemdDatasource{}
)

type healthCheckSystemdDatasource struct {
	healthCheckClient healthcheck_client.HealthCheckClient
}

func NewHealthCheckSystemdDatasource() datasource.DataSource {
	return &healthCheckSystemdDatasource{}
}

func (d *healthCheckSystemdDatasource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_health_check_systemd"
}

func (d *healthCheckSystemdDatasource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {

	var plan *HealthCheckSystemd
	response.Diagnostics.Append(request.Config.Get(ctx, &plan)...)
	if plan == nil {
		response.Diagnostics.AddError("No datasource passed in", "systemd health check is nil")
		return
	}
	for true {
		metrics, getMetricsError := d.healthCheckClient.CheckHttpGet(plan.Address.ValueString(), plan.Path.ValueStringPointer(), plan.TlsEnabled.ValueBoolPointer(), plan.CustomPort.ValueInt64Pointer())

		if getMetricsError != nil {
			response.Diagnostics.AddError("Failed to retrieve metrics for systemd health check", getMetricsError.Error())
			return
		}

		metricLines := strings.Split(*metrics, "\n")

		var serviceMetrics []string
		for _, line := range metricLines {
			if strings.Contains(line, plan.ServiceName.ValueString()) && strings.Contains(line, "systemd") {
				serviceMetrics = append(serviceMetrics, line)
			}
		}

		if len(serviceMetrics) == 0 {
			response.Diagnostics.AddError(fmt.Sprintf("Service %s could not be found", plan.ServiceName.ValueString()), "Check your server configuration and try again")
		}

		for _, metric := range serviceMetrics {
			tflog.Info(ctx, metric)
			metricParts := strings.Split(metric, "}")
			if strings.Contains(metricParts[0], "failed") && strings.TrimSpace(metricParts[1]) == "1" {
				response.Diagnostics.AddError(fmt.Sprintf("The service %s failed to start", plan.ServiceName.ValueString()), "See system logs on the machine for more details")
				return
			} else if strings.Contains(metricParts[0], "active") && strings.TrimSpace(metricParts[1]) == "1" {
				return
			}

		}
		time.Sleep(time.Duration(3) * time.Second)
	}

}

func (d *healthCheckSystemdDatasource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.healthCheckClient = healthcheck_client.NewHealthCheckClient(nil)
}

func (d *healthCheckSystemdDatasource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Checks systemd service status based on the output of a running node exporter instance with the systemd collector enabled",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				Required: true,
			},
			"tls_enabled": schema.BoolAttribute{
				CustomType:  nil,
				Required:    false,
				Optional:    true,
				Computed:    true,
				Sensitive:   false,
				Description: "use https when checking systemd prometheus metrics",
			},
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "name of systemd service",
			},
			"custom_port": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "port that node exporter is running on",
			},
			"path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "http path that node exporter services metrics on",
			},
		},
	}
}
