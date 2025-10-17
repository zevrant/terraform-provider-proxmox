package healthcheck_client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HealthCheckClient interface {
	SetTimeout(timeout int)
	CheckHttpGet(address string, apiPath *string, useTls *bool, requestedPort *int64) (*string, error)
}

type healthCheckClientImpl struct {
	httpClient http.Client
}

func NewHealthCheckClient(timeout *int) HealthCheckClient {
	timeoutValue := 10
	if timeout != nil {
		timeoutValue = *timeout
	}
	client := healthCheckClientImpl{
		httpClient: http.Client{
			Timeout: time.Duration(timeoutValue) * time.Second,
		},
	}
	return &client
}

func (healthCheckClient *healthCheckClientImpl) SetTimeout(timeout int) {
	healthCheckClient.httpClient.Timeout = time.Duration(timeout) * time.Second
}

func (healthCheckClient *healthCheckClientImpl) CheckHttpGet(address string, apiPath *string, useTls *bool, requestedPort *int64) (*string, error) {
	var port, protocol, path string
	if requestedPort == nil {
		port = "80"
	} else {
		port = fmt.Sprintf("%d", *requestedPort)
	}
	if useTls == nil || !*useTls {
		protocol = "http"
	} else {
		protocol = "https"
	}
	if apiPath == nil {
		path = ""
	} else {
		path = *apiPath
	}
	path, _ = strings.CutPrefix(path, "/")
	response, responseError := healthCheckClient.httpClient.Get(fmt.Sprintf("%s://%s:%s/%s", protocol, address, port, path))
	if responseError != nil {
		return nil, responseError
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	responseBodyString := string(responseBody)
	return &responseBodyString, nil
}
