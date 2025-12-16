package proxmox_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) ListNodes() (*proxmoxTypes.NodeListResponse, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes", c.HostURL), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.DoRequest(request, "")

	if responseError != nil {
		return nil, responseError
	}

	var nodeList proxmoxTypes.NodeListResponse

	tflog.Debug(c.Context, fmt.Sprintf("Get Nodes Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &nodeList)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &nodeList, nil
}

func (c *Client) GetNodeNetworkConfig(nodeName string) (*proxmoxTypes.NodeNetworkConfig, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/network", c.HostURL, url.PathEscape(nodeName)), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.DoRequest(request, "")

	if responseError != nil {
		return nil, responseError
	}

	var nodeList proxmoxTypes.NodeNetworkConfig

	tflog.Debug(c.Context, fmt.Sprintf("Get Nodes Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &nodeList)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &nodeList, nil
}
