package proxmox_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
	"strings"
	"time"
)

const FORM_URL_ENCODED = "application/x-www-form-urlencoded"

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
func NewClient(host *string, username *string, password *string, verifyTls *bool, ctx context.Context) *Client {
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

func (c *Client) doRequest(req *http.Request, contentType string) ([]byte, error) {
	return c.doRequestWithResponseStatus(req, http.StatusOK, contentType)
}

func (c *Client) doRequestWithResponseStatus(req *http.Request, expectedResponseStatus int, contentType string) ([]byte, error) {
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
