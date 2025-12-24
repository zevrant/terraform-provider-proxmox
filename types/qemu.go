package types

import "github.com/hashicorp/terraform-plugin-framework/types"

type QemuImage struct {
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	StorageName types.String `tfsdk:"storage_name"`
}

type QemuImageResponse struct {
	Data []QemuImageResponseData `json:"data"`
}

type QemuImageResponseData struct {
	Format  string `json:"format"`
	Volid   string `json:"volid"`
	Content string `json:"content"`
	Ctime   int    `json:"ctime"`
	Size    int    `json:"size"`
}
