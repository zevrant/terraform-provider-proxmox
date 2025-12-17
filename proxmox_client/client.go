package proxmox_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	proxmoxTypes "terraform-provider-proxmox/types"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const FormUrlEncoded = "application/x-www-form-urlencoded"

type ProxmoxClient interface {
	DoRequest(req *http.Request, contentType string) ([]byte, error)
	DoRequestWithResponseStatus(req *http.Request, expectedResponseStatus int, contentType string) ([]byte, error)
	GetTaskStatusByUpid(nodeName *string, upid *string) (*proxmoxTypes.TaskStatus, error)
	UpdateVm(vmCreationBody url.Values, nodeName *string, vmId *string) (*string, error)
	CreateVm(vmCreationBody url.Values, nodeName string) (*string, error)
	GetVmById(nodeName *string, vmId *string) (*proxmoxTypes.QemuResponse, error)
	DeleteVmById(nodeName *string, vmId *string) (*string, error)
	ResizeVmDisk(diskResizeRequest url.Values, nodeName *string, vmId *string) (*string, error)
	GetVmStatus(nodeName *string, vmId *string) (string, error)
	StartVm(nodeName *string, vmId *string) (*string, error)
	ShutdownVm(nodeName *string, vmId *string) (*string, error)
	ListNodes() (*proxmoxTypes.NodeListResponse, error)
	GetNodeNetworkConfig(nodeName string) (*proxmoxTypes.NodeNetworkConfig, error)
	CreateSdnZone(sdnZoneCreationBody url.Values) error
	GetSdnZone(zone string) (*proxmoxTypes.SdnZoneResponse, error)
	DeleteSdnZone(zone string) error
	UpdateSdnZone(sdnZoneCreationBody url.Values) error
	MoveVmDisk(diskName *string, nodeName *string, vmId *string, newStorageName *string) (*string, error)
}

type Client struct {
	HostURL               string
	HTTPClient            *http.Client
	Token                 string
	Auth                  AuthStruct
	EnableTLSVerification bool
	Context               context.Context
}

type AuthStruct struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewClient -
func NewClient(host *string, username *string, password *string, verifyTls *bool, ctx context.Context) ProxmoxClient {
	if host == nil {
		panic("Host Not Provided!!!!")
	}

	if username == nil {
		panic("Username Not Provided!!!!")
	}

	if password == nil {
		panic("Password Not Provided!!!!")
	}

	localTlsVerify := true

	if verifyTls != nil {
		localTlsVerify = *verifyTls
	}

	localHost := fmt.Sprintf("%s/api2/json", *host)
	if !strings.Contains(*host, "https") {
		localHost = fmt.Sprintf("https://%s/api2/json", *host)
	}

	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		// Default Hashicups URL
		HostURL: localHost,
		Auth: AuthStruct{
			Username: *username,
			Password: *password,
		},
		EnableTLSVerification: localTlsVerify,
		Context:               ctx,
	}

	return &c
}

func (c *Client) DoRequest(req *http.Request, contentType string) ([]byte, error) {
	return c.DoRequestWithResponseStatus(req, http.StatusOK, contentType)
}

func (c *Client) DoRequestWithResponseStatus(req *http.Request, expectedResponseStatus int, contentType string) ([]byte, error) {
	tflog.Debug(c.Context, fmt.Sprintf("Making %s request to %s", req.Method, req.URL))
	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.Auth.Username, c.Auth.Password))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	if !c.EnableTLSVerification {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tflog.Debug(c.Context, fmt.Sprintf("status code was %d for url %s", res.StatusCode, req.URL.Path))
	if res.StatusCode != expectedResponseStatus {
		tflog.Error(c.Context, fmt.Sprintf("statusCode: %d, status:%s, body: %s", res.StatusCode, res.Status, body))
		return body, fmt.Errorf("status: %s, body: %s", res.Status, body)
	}

	return body, err
}
