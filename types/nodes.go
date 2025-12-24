package types

type NodeListResponse struct {
	Data []NodeResponse `json:"data"`
}

type NodeResponse struct {
	Status         string  `json:"status"`
	MaxMem         int64   `json:"maxmem"`
	Level          string  `json:"level"`
	Id             string  `json:"id"`
	Type           string  `json:"type"`
	MaxCpu         int     `json:"maxcpu"`
	Maxdisk        int64   `json:"maxdisk"`
	Node           string  `json:"node"`
	Uptime         int64   `json:"uptime"`
	Cpu            float64 `json:"cpu"`
	Disk           int64   `json:"disk"`
	Mem            int64   `json:"mem"`
	SslFingerprint string  `json:"ssl_fingerprint"`
}

type NodeNetworkConfig struct {
	Data []struct {
		Method      string   `json:"method"`
		Exists      int      `json:"exists,omitempty"`
		Families    []string `json:"families"`
		Method6     string   `json:"method6"`
		Type        string   `json:"type"`
		Priority    int      `json:"priority"`
		Iface       string   `json:"iface"`
		Active      int      `json:"active,omitempty"`
		Address     string   `json:"address,omitempty"`
		Cidr        string   `json:"cidr,omitempty"`
		BridgePorts string   `json:"bridge_ports,omitempty"`
		Autostart   int      `json:"autostart,omitempty"`
		Gateway     string   `json:"gateway,omitempty"`
		BridgeFd    string   `json:"bridge_fd,omitempty"`
		Netmask     string   `json:"netmask,omitempty"`
		BridgeStp   string   `json:"bridge_stp,omitempty"`
	} `json:"data"`
}

type NodeStorageResponse struct {
	Data []NodeStorageResponseItem `json:"data"`
}

type NodeStorageResponseItem struct {
	Enabled      int     `json:"enabled"`
	Used         int64   `json:"used"`
	Content      string  `json:"content"`
	Avail        int64   `json:"avail"`
	Shared       int     `json:"shared"`
	Type         string  `json:"type"`
	Storage      string  `json:"storage"`
	Total        int64   `json:"total"`
	Active       int     `json:"active"`
	UsedFraction float64 `json:"used_fraction,omitempty"`
}
