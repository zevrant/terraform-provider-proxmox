package types

type TaskStatus struct {
	Data struct {
		Tokenid    string `json:"tokenid"`
		Upid       string `json:"upid"`
		Pstart     int    `json:"pstart"`
		User       string `json:"user"`
		Pid        int    `json:"pid"`
		Status     string `json:"status"`
		Node       string `json:"node"`
		Id         string `json:"id"`
		Starttime  int    `json:"starttime"`
		Exitstatus string `json:"exitstatus"`
		Type       string `json:"type"`
	} `json:"data"`
}

type TaskCreationResponse struct {
	Upid string `json:"data"`
}
