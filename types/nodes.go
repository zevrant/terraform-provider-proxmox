package types

type NodeListResponse struct {
	Data []struct {
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
	} `json:"data"`
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
