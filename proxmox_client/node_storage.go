package proxmox_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (c *Client) ListStorageContent(nodeName *string, storageName *string) (*proxmoxTypes.QemuImageResponse, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/storage/%s/content", c.HostURL, *nodeName, *storageName), nil)

	if requestCreationError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to create list storage content request: %s", requestCreationError.Error()))
		return nil, requestCreationError
	}

	body, responseError := c.DoRequest(request, FormUrlEncoded)

	if responseError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to List Storage contents %s, on node %s: %s", *storageName, *nodeName, responseError.Error()))
		return nil, responseError
	}

	var images proxmoxTypes.QemuImageResponse
	unmarshallingError := json.Unmarshal(body, &images)

	if unmarshallingError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to unmarshal list storage content response: %s", unmarshallingError.Error()))
		return nil, unmarshallingError
	}

	return &images, nil
}

func (c *Client) ListStorageDestinations(nodeName *string) (*proxmoxTypes.NodeStorageResponse, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/storage/", c.HostURL, *nodeName), nil)

	if requestCreationError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to create list node storage request : %s", requestCreationError.Error()))
		return nil, requestCreationError
	}

	body, responseError := c.DoRequest(request, FormUrlEncoded)

	if responseError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to List Storage on node %s: %s", *nodeName, responseError.Error()))
		return nil, responseError
	}

	var storageList proxmoxTypes.NodeStorageResponse
	unmarshallingError := json.Unmarshal(body, &storageList)

	if unmarshallingError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to unmarshal list node storage response: %s", unmarshallingError.Error()))
		return nil, unmarshallingError
	}

	return &storageList, nil
}
