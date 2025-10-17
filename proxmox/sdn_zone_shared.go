package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"terraform-provider-proxmox/proxmox_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sdnZone struct {
	Zone  types.String `tfsdk:"zone"`
	Ipam  types.String `tfsdk:"ipam"`
	Type  types.String `tfsdk:"type"`
	Nodes types.List   `tfsdk:"nodes"`
	Peers types.List   `tfsdk:"peers"`
}

func updateSdnZoneFromResponse(zone *sdnZone, tfContext context.Context, response proxmox_client.SdnZoneResponse) {
	nodes, _ := types.ListValueFrom(tfContext, types.StringType, strings.Split(response.Data.Nodes, ","))
	peers, _ := types.ListValueFrom(tfContext, types.StringType, strings.Split(strings.Trim(response.Data.Peers, " "), ","))
	zone.Zone = types.StringValue(response.Data.Zone)
	zone.Ipam = types.StringValue(response.Data.Ipam)
	zone.Type = types.StringValue(response.Data.Type)
	zone.Nodes = nodes
	zone.Peers = peers
}

func assembleCreateSdnZoneRequest(request *url.Values, sdnZone sdnZone, tfContext context.Context) {
	peersList := make([]types.String, 0, len(sdnZone.Peers.Elements()))
	_ = sdnZone.Peers.ElementsAs(tfContext, &peersList, false)
	peers := ""
	for _, peer := range peersList {
		if peers == "" {
			peers += peer.ValueString()
		} else {
			peers = fmt.Sprintf("%s,%s", peers, peer.ValueString())
		}
	}

	nodesList := make([]types.String, 0, len(sdnZone.Nodes.Elements()))
	_ = sdnZone.Nodes.ElementsAs(tfContext, &nodesList, false)

	nodes := ""
	for _, node := range nodesList {
		if nodes == "" {
			nodes += node.ValueString()
		}
		nodes += fmt.Sprintf(",%s", node.ValueString())
	}
	request.Add("type", sdnZone.Type.ValueString())
	request.Add("zone", sdnZone.Zone.ValueString())
	request.Add("peers", peers)
	request.Add("ipam", sdnZone.Ipam.ValueString())
	request.Add("nodes", nodes)
}
