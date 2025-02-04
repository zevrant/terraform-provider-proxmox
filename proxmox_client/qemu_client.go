package proxmox_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
	"net/url"
)

func (c *Client) GetVmById(nodeName string, vmId string) (*QemuResponse, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/qemu/%s/config", c.HostURL, nodeName, vmId), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, "")

	if responseError != nil {
		return nil, responseError
	}

	var qemuResponse QemuResponse

	tflog.Debug(c.Context, fmt.Sprintf("Get Vm Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &qemuResponse)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	var otherFields map[string]interface{}
	unmarshallingError = json.Unmarshal(body, &otherFields)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	qemuResponse.Data.OtherFields = otherFields["data"].(map[string]interface{})
	tflog.Debug(c.Context, "VM Gen ID is "+qemuResponse.Data.VmGenId)
	return &qemuResponse, nil
}

func (c *Client) CreateVm(vmCreationBody url.Values, nodeName string) (*string, error) {

	request, requestCreationError := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/nodes/%s/qemu", c.HostURL, nodeName), bytes.NewBufferString(vmCreationBody.Encode()))

	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)
	if responseError != nil {
		return nil, errors.Join(responseError, errors.New(string(body)))
	}

	var qemuCreationResponse = TaskCreationResponse{}

	unmarshallingError := json.Unmarshal(body, &qemuCreationResponse)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &qemuCreationResponse.Upid, nil
}

func (c *Client) UpdateVm(vmCreationBody url.Values, nodeName string, vmId string) (*string, error) {

	tflog.Debug(c.Context, fmt.Sprintf("Request Body: %s", vmCreationBody.Encode()))

	request, requestCreationError := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/nodes/%s/qemu/%s/config", c.HostURL, nodeName, vmId), bytes.NewBufferString(vmCreationBody.Encode()))

	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)
	if responseError != nil {
		return nil, errors.Join(responseError, errors.New(string(body)))
	}

	var qemuCreationResponse = TaskCreationResponse{}

	unmarshallingError := json.Unmarshal(body, &qemuCreationResponse)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &qemuCreationResponse.Upid, nil
}

func (c *Client) GetTaskStatusByUpid(nodeName string, upid string) (*TaskStatus, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/tasks/%s/status", c.HostURL, nodeName, upid), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)

	if responseError != nil {
		return nil, responseError
	}

	var taskStatus TaskStatus

	unmarshallingError := json.Unmarshal(body, &taskStatus)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &taskStatus, nil
}

func (c *Client) DeleteVmById(nodeName string, vmId string) (*string, error) {
	params := url.Values{}
	params.Add("destroy-unreferenced-disks", "1")
	params.Add("purge", "1")

	request, requestCreationError := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/nodes/%s/qemu/%s?%s", c.HostURL, nodeName, vmId, params.Encode()), nil)
	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)

	if responseError != nil {
		return nil, responseError
	}

	var deleteVmResponse TaskCreationResponse

	tflog.Debug(c.Context, fmt.Sprintf("Get Vm Response is %s", string(body)))

	unmarshallingError := json.Unmarshal(body, &deleteVmResponse)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &deleteVmResponse.Upid, nil
}

func (c *Client) ResizeVmDisk(diskResizeRequest url.Values, nodeName string, vmId string) (*string, error) {
	tflog.Debug(c.Context, diskResizeRequest.Encode())
	request, requestCreationError := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/nodes/%s/qemu/%s/resize", c.HostURL, nodeName, vmId), bytes.NewBufferString(diskResizeRequest.Encode()))

	if requestCreationError != nil {
		return nil, requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)
	if responseError != nil {
		return nil, errors.Join(responseError, errors.New(string(body)))
	}

	var diskResizeResponse = TaskCreationResponse{}

	unmarshallingError := json.Unmarshal(body, &diskResizeResponse)
	if unmarshallingError != nil {
		return nil, unmarshallingError
	}

	return &diskResizeResponse.Upid, nil
}

func (c *Client) GetVmStatus(nodeName string, vmId string) (string, error) {
	request, requestCreationError := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/nodes/%s/qemu/%s/status/current", c.HostURL, nodeName, vmId), nil)

	if requestCreationError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to create GetVmStatus http request: %s", requestCreationError.Error()))
		return "", requestCreationError
	}

	body, responseError := c.doRequest(request, "application/json")

	if responseError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to get status of VM %s, from node %s: %s", vmId, nodeName, responseError.Error()))
		return "", responseError
	}

	var vmStatus = VmStatus{}
	unmarshallingError := json.Unmarshal(body, &vmStatus)

	if unmarshallingError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to unmarshal get vm status response: %s", unmarshallingError.Error()))
		return "", unmarshallingError
	}

	return vmStatus.Data.Status, nil
}

func (c *Client) StartVm(nodeName string, vmId string) (string, error) {
	request, requestCreationError := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/nodes/%s/qemu/%s/status/start", c.HostURL, nodeName, vmId), nil)

	if requestCreationError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to create Start Vm Request http request: %s", requestCreationError.Error()))
		return "", requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)

	if responseError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to start VM %s, on node %s: %s", vmId, nodeName, responseError.Error()))
		return "", responseError
	}

	var vmStatus = TaskCreationResponse{}
	unmarshallingError := json.Unmarshal(body, &vmStatus)

	if unmarshallingError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to unmarshal start vm response: %s", unmarshallingError.Error()))
		return "", unmarshallingError
	}

	return vmStatus.Upid, nil
}

func (c *Client) ShutdownVm(nodeName string, vmId string) (string, error) {
	request, requestCreationError := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/nodes/%s/qemu/%s/status/shutdown", c.HostURL, nodeName, vmId), nil)

	if requestCreationError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to create Shutdown Vm Request http request: %s", requestCreationError.Error()))
		return "", requestCreationError
	}

	body, responseError := c.doRequest(request, FORM_URL_ENCODED)

	if responseError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to shutdown VM %s, on node %s: %s", vmId, nodeName, responseError.Error()))
		return "", responseError
	}

	var vmStatus = TaskCreationResponse{}
	unmarshallingError := json.Unmarshal(body, &vmStatus)

	if unmarshallingError != nil {
		tflog.Error(c.Context, fmt.Sprintf("Failed to unmarshal shutdown vm response: %s", unmarshallingError.Error()))
		return "", unmarshallingError
	}

	return vmStatus.Upid, nil
}
