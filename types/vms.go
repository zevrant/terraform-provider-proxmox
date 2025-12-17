package types

import "github.com/hashicorp/terraform-plugin-framework/types"

const NetworkInterfaceTypes = "e1000 | e1000-82540em | e1000-82544gc | e1000-82545em | e1000e | i82551 | i82557b | i82559er | ne2k_isa | ne2k_pci | pcnet | rtl8139 | virtio | vmxnet3"

type VmModel struct {
	Acpi                 types.Bool           `tfsdk:"acpi"`
	Agent                types.Bool           `tfsdk:"qemu_agent_enabled"`
	Bios                 types.String         `tfsdk:"bios"`
	BootOrder            types.List           `tfsdk:"boot_order"`
	CloudInitUpgrade     types.Bool           `tfsdk:"perform_cloud_init_upgrade"`
	Cores                types.Int64          `tfsdk:"cores"`
	Cpu                  types.String         `tfsdk:"cpu_type"`
	CpuLimit             types.Int64          `tfsdk:"cpu_limit"`
	Description          types.String         `tfsdk:"description"`
	Disks                []VmDisk             `tfsdk:"disk"`
	HostStartupOrder     types.Int64          `tfsdk:"host_startup_order"`
	IpConfigurations     []VmIpConfig         `tfsdk:"ip_config"`
	Kvm                  types.Bool           `tfsdk:"kvm"`
	Memory               types.Int64          `tfsdk:"memory"`
	Name                 types.String         `tfsdk:"name"`
	Nameserver           types.String         `tfsdk:"nameserver"`
	NetworkInterfaces    []VmNetworkInterface `tfsdk:"network_interface"`
	NodeName             types.String         `tfsdk:"node_name"`
	Numa                 types.Bool           `tfsdk:"numa_active"`
	OnBoot               types.Bool           `tfsdk:"start_on_boot"`
	OsType               types.String         `tfsdk:"os_type"`
	Protection           types.Bool           `tfsdk:"protection"`
	ScsiHw               types.String         `tfsdk:"scsi_hw"`
	Sockets              types.Int64          `tfsdk:"sockets"`
	SshKeys              types.List           `tfsdk:"ssh_keys"`
	Tags                 types.List           `tfsdk:"tags"`
	VmGenId              types.String         `tfsdk:"vmgenid"`
	VmId                 types.String         `tfsdk:"vm_id"`
	DefaultUser          types.String         `tfsdk:"default_user"`
	CloudInitStorageName types.String         `tfsdk:"cloud_init_storage_name"`
	PowerState           types.String         `tfsdk:"power_state"`
}

type QemuResponse struct { //some of the optional fields in the spec will not appear if they do not differ from the default value
	Data struct {
		Acpi             int                    `json:"acpi"`
		AutoStart        int                    `json:"autostart"`
		Bios             string                 `json:"bios"`
		CpuLimit         string                 `json:"cpulimit"`
		HostStartupOrder string                 `json:"startup"`
		Kvm              int                    `json:"kvm"`
		Tags             string                 `json:"tags"`
		Memory           string                 `json:"memory"`
		Name             string                 `json:"name"`
		Cpu              string                 `json:"cpu"`
		OnBoot           int                    `json:"onboot"`
		Description      string                 `json:"description"`
		OsType           string                 `json:"ostype"`
		VmGenId          string                 `json:"vmgenid"`
		Sockets          int                    `json:"sockets"`
		Meta             string                 `json:"meta"`
		ScsiHw           string                 `json:"scsihw"`
		Agent            string                 `json:"agent"`
		Boot             string                 `json:"boot"`
		Cores            int                    `json:"cores"`
		Numa             int                    `json:"numa"`
		Nameserver       string                 `json:"nameserver"`
		CloudInitUpgrade int                    `json:"ciupgrade"`
		Protection       int                    `json:"protection"`
		SshKeys          string                 `json:"sshKeys"`
		CiUser           string                 `json:"ciuser"`
		OtherFields      map[string]interface{} `json:"-"` //skip this key
	} `json:"data"`
}

type VmStatusData struct {
	Name   string `json:"name"`
	Agent  int    `json:"agent"`
	Status string `json:"status"`
}

type VmStatus struct {
	Data VmStatusData `json:"data"`
}

type VmDisk struct {
	Id              types.Int64  `tfsdk:"id"`
	BusType         types.String `tfsdk:"bus_type"`
	StorageLocation types.String `tfsdk:"storage_location"`
	IoThread        types.Bool   `tfsdk:"io_thread"`
	Size            types.String `tfsdk:"size"`
	Cache           types.String `tfsdk:"cache"`
	AsyncIo         types.String `tfsdk:"async_io"`
	Replicate       types.Bool   `tfsdk:"replicate"`
	ReadOnly        types.Bool   `tfsdk:"read_only"`
	SsdEmulation    types.Bool   `tfsdk:"ssd_emulation"`
	Backup          types.Bool   `tfsdk:"backup_enabled"`
	Discard         types.Bool   `tfsdk:"discard_enabled"`
	Order           types.Int64  `tfsdk:"order"`
	ImportFrom      types.String `tfsdk:"import_from"`
	Path            types.String `tfsdk:"import_path"`
}

type VmNetworkInterface struct {
	Type       types.String `tfsdk:"type"`
	MacAddress types.String `tfsdk:"mac_address"`
	Bridge     types.String `tfsdk:"bridge"`
	Firewall   types.Bool   `tfsdk:"firewall"`
	Order      types.Int64  `tfsdk:"order"`
	Mtu        types.Int64  `tfsdk:"mtu"`
}

type VmIpConfig struct {
	IpAddress types.String `tfsdk:"ip_address"`
	Gateway   types.String `tfsdk:"gateway"`
	Order     types.Int64  `tfsdk:"order"`
}

type DiskChanges struct {
	ToBeAdded    []VmDisk
	ToBeUpdated  []VmDisk
	ToBeRemove   []VmDisk
	ToBeResized  []VmDisk
	ToBeMigrated map[VmDisk]VmDisk
}
