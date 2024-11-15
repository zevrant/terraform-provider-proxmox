package proxmox_client

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
	"net/url"
)

func (c *Client) ListNodes() (*NodeListResponse, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes", c.HostURL), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, "")

	if responseError != nil {
		return nil, responseError
	}

	var nodeList NodeListResponse

	tflog.Debug(c.Context, fmt.Sprintf("Get Nodes Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &nodeList)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &nodeList, nil
}

func (c *Client) GetNodeNetworkConfig(nodeName string) (*NodeNetworkConfig, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/network", c.HostURL, url.PathEscape(nodeName)), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, "")

	if responseError != nil {
		return nil, responseError
	}

	var nodeList NodeNetworkConfig

	tflog.Debug(c.Context, fmt.Sprintf("Get Nodes Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &nodeList)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &nodeList, nil
}
