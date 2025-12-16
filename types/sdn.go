package types

type SdnZoneResponse struct {
	Data struct {
		Digest string `json:"digest"`
		Zone   string `json:"zone"`
		Ipam   string `json:"ipam"`
		Nodes  string `json:"nodes"`
		Peers  string `json:"peers"`
		Type   string `json:"type"`
	} `json:"data"`
}
