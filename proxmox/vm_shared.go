package proxmox

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-proxmox/proxmox_client"
	"time"
)

const NETWORK_INTERFACE_TYPES = "e1000 | e1000-82540em | e1000-82544gc | e1000-82545em | e1000e | i82551 | i82557b | i82559er | ne2k_isa | ne2k_pci | pcnet | rtl8139 | virtio | vmxnet3"

type VmModel struct {
	Acpi              types.Bool           `tfsdk:"acpi"`
	Agent             types.Bool           `tfsdk:"qemu_agent_enabled"`
	AutoStart         types.Bool           `tfsdk:"auto_start"`
	Bios              types.String         `tfsdk:"bios"`
	BootOrder         types.List           `tfsdk:"boot_order"`
	CloudInitUpgrade  types.Bool           `tfsdk:"perform_cloud_init_upgrade"`
	Cores             types.Int64          `tfsdk:"cores"`
	Cpu               types.String         `tfsdk:"cpu_type"`
	CpuLimit          types.Int64          `tfsdk:"cpu_limit"`
	Description       types.String         `tfsdk:"description"`
	Disks             []VmDisk             `tfsdk:"disks"`
	HostStartupOrder  types.Int64          `tfsdk:"host_startup_order"`
	IpConfigurations  []VmIpConfig         `tfsdk:"ip_configs"`
	Kvm               types.Bool           `tfsdk:"kvm"`
	Memory            types.Int64          `tfsdk:"memory"`
	Name              types.String         `tfsdk:"name"`
	Nameserver        types.String         `tfsdk:"nameserver"`
	NetworkInterfaces []VmNetworkInterface `tfsdk:"network_interfaces"`
	NodeName          types.String         `tfsdk:"node_name"`
	Numa              types.Bool           `tfsdk:"numa_active"`
	OnBoot            types.Bool           `tfsdk:"start_on_boot"`
	OsType            types.String         `tfsdk:"os_type"`
	Protection        types.Bool           `tfsdk:"protection"`
	ScsiHw            types.String         `tfsdk:"scsi_hw"`
	Sockets           types.Int64          `tfsdk:"sockets"`
	SshKeys           types.List           `tfsdk:"ssh_keys"`
	Tags              types.List           `tfsdk:"tags"`
	VmGenId           types.String         `tfsdk:"vmgenid"`
	VmId              types.String         `tfsdk:"vm_id"`
}

type VmNetworkInterface struct {
	Type       types.String `tfsdk:"type"`
	MacAddress types.String `tfsdk:"mac_address"`
	Bridge     types.String `tfsdk:"bridge"`
	Firewall   types.Bool   `tfsdk:"firewall"`
	Order      types.Int64  `tfsdk:"order"`
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
}

type VmIpConfig struct {
	IpAddress types.String `tfsdk:"ip_address"`
	Gateway   types.String `tfsdk:"gateway"`
	Order     types.Int64  `tfsdk:"order"`
}

func updateVmModelFromResponse(vmModel *VmModel, response proxmox_client.QemuResponse, tfContext *context.Context) {
	memory, _ := strconv.ParseInt(response.Data.Memory, 10, 64)
	tags := strings.Split(strings.Trim(response.Data.Tags, " "), ";")

	if response.Data.Tags == " " {
		tags = []string{}
	}

	vmModel.Cpu = types.StringValue(response.Data.Cpu)
	tflog.Debug(*tfContext, fmt.Sprintf("Setting cpu type to %s", response.Data.Cpu))
	vmModel.Memory = types.Int64Value(memory)
	vmModel.Tags, _ = types.ListValueFrom(*tfContext, types.StringType, tags)
	vmModel.Name = types.StringValue(response.Data.Name)
	vmModel.OnBoot = types.BoolValue(response.Data.OnBoot == 1)
	vmModel.Description = types.StringValue(response.Data.Description)
	vmModel.VmGenId = types.StringValue(response.Data.VmGenId)
	vmModel.Sockets = types.Int64Value(int64(response.Data.Sockets))
	vmModel.OsType = types.StringValue(response.Data.OsType)
	vmModel.ScsiHw = types.StringValue(response.Data.ScsiHw)
	vmModel.Agent = types.BoolValue(response.Data.Agent == "1") //No clue why this is coming back a string as opposed to an int like the others
	vmModel.BootOrder, _ = types.ListValueFrom(*tfContext, types.StringType, strings.Split(strings.Replace(response.Data.Boot, "order=", "", 1), ","))
	vmModel.Numa = types.BoolValue(response.Data.Numa == 1)
	vmModel.Cores = types.Int64Value(int64(response.Data.Cores))
	vmModel.Acpi = types.BoolValue(response.Data.Acpi == 1)
	cpuLimit, _ := strconv.ParseInt(response.Data.CpuLimit, 10, 64)
	vmModel.CpuLimit = types.Int64Value(cpuLimit)
	vmModel.Kvm = types.BoolValue(response.Data.Kvm == 1)
	vmModel.Nameserver = types.StringValue(response.Data.Nameserver)
	vmModel.CloudInitUpgrade = types.BoolValue(response.Data.CloudInitUpgrade == 1)
	vmModel.Protection = types.BoolValue(response.Data.Protection != 0)
	//vmModel.SshKeys = TODO: finish implementation
	startupOrder, _ := strconv.ParseInt(strings.Replace(response.Data.HostStartupOrder, "order=", "", 1), 10, 64)

	vmModel.HostStartupOrder = types.Int64Value(startupOrder)
	vmModel.AutoStart = types.BoolValue(response.Data.AutoStart == 1)
	if response.Data.Bios == "" {
		vmModel.Bios = types.StringValue("seabios")
	} else {
		vmModel.Bios = types.StringValue(response.Data.Bios)
	}
	vmModel.Disks = mapDisksFromQemuResponse(response.Data.OtherFields)
	vmModel.NetworkInterfaces = mapNetworkInterfacesFromQemuResponse(response.Data.OtherFields)
	vmModel.IpConfigurations = mapIpConfigsFromQemuResponse(response.Data.OtherFields)
}

func mapKeyValuePairsToMap(pairs []string) map[string]string {
	mappedPairs := make(map[string]string)

	for _, pair := range pairs {
		if strings.Contains(pair, "=") {
			splitPair := strings.Split(pair, "=")
			mappedPairs[splitPair[0]] = splitPair[1]
		} else {
			fmt.Println(fmt.Sprintf("Pair didn't contain '=': %s", pair))
		}
	}

	return mappedPairs
}

func mapDisksFromQemuResponse(otherFields map[string]interface{}) []VmDisk {
	var vmDisks []VmDisk
	var keySlice []string = getDiskKeysFromJsonDict(otherFields)

	for order, key := range keySlice {
		disk := otherFields[key].(string)
		diskParts := strings.Split(disk, ",")
		diskFieldMap := mapKeyValuePairsToMap(diskParts[1:])
		cache := diskFieldMap["cache"]

		if cache == "" {
			cache = "default"
		}

		diskNumber, _ := strconv.ParseInt(strings.Split(strings.Split(diskParts[0], ":")[1], "-")[3], 10, 64)
		if strings.Contains(diskParts[0], "cloudinit") {
			continue
		}
		newVmDisk := VmDisk{
			Id:              types.Int64Value(diskNumber),
			BusType:         types.StringValue(key[:len("scsi")]),
			StorageLocation: types.StringValue(strings.Split(diskParts[0], ":")[0]),
			IoThread:        types.BoolValue(diskFieldMap["iothread"] == "1"),
			Size:            types.StringValue(diskFieldMap["size"]),
			Cache:           types.StringValue(cache),
			AsyncIo:         types.StringValue(diskFieldMap["aio"]),
			Replicate:       types.BoolValue(diskFieldMap["replicate"] == ""),
			ReadOnly:        types.BoolValue(diskFieldMap["ro"] == "1"),
			SsdEmulation:    types.BoolValue(diskFieldMap["ssd"] == "1"),
			Backup:          types.BoolValue(diskFieldMap["backup"] == ""),
			Discard:         types.BoolValue(diskFieldMap["discard"] == "on"),
			Order:           types.Int64Value(int64(order)),
		}

		if newVmDisk.AsyncIo.ValueString() == "" {
			newVmDisk.AsyncIo = types.StringValue("default")
		}
		vmDisks = append(vmDisks, newVmDisk)
	}
	return vmDisks
}

func mapNetworkInterfacesFromQemuResponse(otherFields map[string]interface{}) []VmNetworkInterface {
	var vmNics []VmNetworkInterface
	var keySlice []string
	for key, _ := range otherFields {
		matched, _ := regexp.MatchString("net\\d+", key)
		if matched {
			keySlice = append(keySlice, key)
		}
	}

	sort.Strings(keySlice)
	var networkInterfaceTypes []string = strings.Split(NETWORK_INTERFACE_TYPES, " | ")
	for order, key := range keySlice {
		if !strings.Contains(key, "net") {
			continue
		}

		nic := otherFields[key].(string)

		nicParts := strings.Split(nic, ",")

		var networkInterfaceType = ""

		mappedNicFields := mapKeyValuePairsToMap(nicParts)

		for netInterfaceKey, _ := range mappedNicFields {
			if slices.Contains(networkInterfaceTypes, netInterfaceKey) {
				networkInterfaceType = netInterfaceKey
			}
		}

		newVmNic := VmNetworkInterface{
			MacAddress: types.StringValue(mappedNicFields[networkInterfaceType]),
			Bridge:     types.StringValue(mappedNicFields["bridge"]),
			Firewall:   types.BoolValue(mappedNicFields["firewall"] == "1"),
			Order:      types.Int64Value(int64(order)),
			Type:       types.StringValue(networkInterfaceType),
		}

		vmNics = append(vmNics, newVmNic)
	}
	return vmNics
}

func mapIpConfigsFromQemuResponse(otherFields map[string]interface{}) []VmIpConfig {
	var vmIpConfigs []VmIpConfig
	var keySlice []string
	for key, _ := range otherFields {
		matched, _ := regexp.MatchString("ipconfig\\d+", key)
		if matched {
			keySlice = append(keySlice, key)
		}
	}

	sort.Strings(keySlice)
	fmt.Println(fmt.Sprintf("There are %d ip configurations to be loaded", len(keySlice)))
	for order, key := range keySlice {
		fmt.Println(fmt.Sprintf("Loading ipconfig %s", key))
		nic := otherFields[key].(string)

		nicParts := strings.Split(nic, ",")

		mappedIpConfigFields := mapKeyValuePairsToMap(nicParts)

		newVmNic := VmIpConfig{
			IpAddress: types.StringValue(mappedIpConfigFields["ip"]),
			Gateway:   types.StringValue(mappedIpConfigFields["gw"]),
			Order:     types.Int64Value(int64(order)),
		}

		vmIpConfigs = append(vmIpConfigs, newVmNic)
	}
	return vmIpConfigs
}

func createVmRequest(vmModel *VmModel, tfContext *context.Context, cloudInitEnabled bool, createNew bool) url.Values {
	params := url.Values{}
	params.Add("vmid", vmModel.VmId.String())
	params.Add("name", vmModel.Name.ValueString())

	tagsList := make([]types.String, 0, len(vmModel.Tags.Elements()))
	_ = vmModel.Tags.ElementsAs(*tfContext, &tagsList, false)
	tags := ""
	for _, tag := range tagsList {
		if tags == "" {
			tags = tag.ValueString()
		} else {
			tags += "," + tag.ValueString()
		}
	}

	var bootOrder = ""
	disks := make([]types.String, 0, len(vmModel.BootOrder.Elements()))
	_ = vmModel.BootOrder.ElementsAs(*tfContext, &disks, false)
	for _, disk := range disks {
		if bootOrder == "" {
			bootOrder = disk.ValueString()
		} else {
			bootOrder += ";" + disk.ValueString()
		}
	}

	sshKeys := ""
	keysList := make([]types.String, 0, len(vmModel.SshKeys.Elements()))

	for _, key := range keysList {
		sshKeys += fmt.Sprintf("\n%s", key)
	}

	params.Add("acpi", mapBoolToProxmoxString(vmModel.Acpi.ValueBool()))
	params.Add("agent", mapBoolToProxmoxString(vmModel.Agent.ValueBool()))
	params.Add("autostart", mapBoolToProxmoxString(vmModel.AutoStart.ValueBool()))
	params.Add("bios", vmModel.Bios.ValueString())
	params.Add("boot", fmt.Sprintf("order=%s", bootOrder))
	params.Add("ciupgrade", mapBoolToProxmoxString(vmModel.CloudInitUpgrade.ValueBool()))
	params.Add("cpu", vmModel.Cpu.ValueString())
	params.Add("hotplug", "network,usb")
	params.Add("cpulimit", vmModel.CpuLimit.String())
	params.Add("description", vmModel.Description.ValueString())
	for index, ipConfig := range vmModel.IpConfigurations {
		params.Add(fmt.Sprintf("ipconfig%d", index), fmt.Sprintf("gw=%s,ip=%s", ipConfig.Gateway.ValueString(), ipConfig.IpAddress.ValueString()))
	}
	params.Add("kvm", mapBoolToProxmoxString(vmModel.Kvm.ValueBool()))
	params.Add("memory", vmModel.Memory.String())
	params.Add("nameserver", vmModel.Nameserver.ValueString())
	params.Add("scsihw", vmModel.ScsiHw.ValueString())
	params.Add("sockets", vmModel.Sockets.String())
	if sshKeys != "" {
		params.Add("sshkeys", sshKeys)
	}
	params.Add("cores", vmModel.Cores.String())
	params.Add("tags", tags)
	params.Add("startup", fmt.Sprintf("order=%d", vmModel.HostStartupOrder.ValueInt64()))
	params.Add("protection", mapBoolToProxmoxString(vmModel.Protection.ValueBool()))
	params.Add("ostype", vmModel.OsType.ValueString())
	if createNew {
		attachVmDiskRequests(vmModel.Disks, &params, vmModel.VmId.ValueString(), cloudInitEnabled, createNew)
	}
	attachVmNicRequests(*vmModel, &params)
	return params
}

func attachVmDiskRequests(disks []VmDisk, params *url.Values, vmId string, cloudInitEnabled bool, createNew bool) {
	for _, disk := range disks {
		//local-zfs:vm-140-disk-0,aio=io_uring,backup=0,cache=directsync,discard=on,iothread=1,replicate=0,ro=1,size=32G,ssd=1
		var diskString string
		if createNew {
			diskString = fmt.Sprintf("%s:%s,size=%s", disk.StorageLocation.ValueString(), disk.Size.ValueString()[:len(disk.Size.ValueString())-1], disk.Size.ValueString())
		} else {
			diskString = fmt.Sprintf("%s:vm-%s-disk-%d,size=%s", disk.StorageLocation.ValueString(), vmId, disk.Id.ValueInt64(), disk.Size.ValueString())
		}

		if disk.AsyncIo.ValueString() != "default" {
			diskString = fmt.Sprintf("%s,aio=%s", diskString, disk.AsyncIo.ValueString())
		}
		if !disk.Backup.ValueBool() {
			diskString = fmt.Sprintf("%s,backup=0", diskString)
		}
		if disk.Cache.ValueString() != "default" {
			diskString = fmt.Sprintf("%s,cache=%s", diskString, disk.Cache.ValueString())
		}
		if disk.Discard.ValueBool() {
			diskString = fmt.Sprintf("%s,discard=on", diskString)
		}
		diskString = fmt.Sprintf("%s,iothread=%s", diskString, mapBoolToProxmoxString(disk.IoThread.ValueBool()))
		if !disk.Replicate.ValueBool() {
			diskString = fmt.Sprintf("%s,replicate=0", diskString)
		}
		if disk.ReadOnly.ValueBool() {
			diskString = fmt.Sprintf("%s,ro=1", diskString)
		}
		if disk.SsdEmulation.ValueBool() {
			diskString = fmt.Sprintf("%s,ssd=1", diskString)
		}
		params.Add(disk.BusType.ValueString()+disk.Order.String(), diskString)
	}
	if cloudInitEnabled {
		params.Add(fmt.Sprintf("scsi%d", len(disks)), "local-zfs:cloudinit,media=cdrom")
	}
}

func attachVmNicRequests(vmModel VmModel, params *url.Values) {
	for _, nicConfig := range vmModel.NetworkInterfaces {
		params.Add(fmt.Sprintf("net%d", nicConfig.Order.ValueInt64()), fmt.Sprintf("%s=%s,bridge=%s,firewall=%s", nicConfig.Type.ValueString(), nicConfig.MacAddress.ValueString(), nicConfig.Bridge.ValueString(), mapBoolToProxmoxString(nicConfig.Firewall.ValueBool())))
	}
}

/**
 * @description proxmox uses the string value of 1 for true and 0 for false when dealing with booleans in their api
 * @param aBooleanValue: the value to be converted to strings used by proxmox
 *
 * @return 1 when true  0 when false
 */
func mapBoolToProxmoxString(aBooleanValue bool) string {
	if aBooleanValue {
		return "1"
	}
	return "0"
}

func mapProxmoxStringToBool(proxmoxBoolString string) bool {
	return proxmoxBoolString == "1"
}

func waitForTaskCompletion(nodeName string, taskUpid string, client *proxmox_client.Client) error {
	for {

		taskStatus, getTaskStatusError := client.GetTaskStatusByUpid(nodeName, taskUpid)

		if getTaskStatusError != nil {
			return getTaskStatusError
		}

		if taskStatus.Data.Status == "stopped" && taskStatus.Data.Exitstatus != "OK" {
			return errors.New(fmt.Sprintf("proxmox failed to complete %s task, please check logs in the proxmox console for more details", taskStatus.Data.Type))
		} else if taskStatus.Data.Status == "stopped" && taskStatus.Data.Exitstatus == "OK" {
			break
		}
		time.Sleep(3 * time.Second)
	}
	return nil
}

func getDiskKeysFromJsonDict(dict map[string]interface{}) []string {
	var keySlice []string
	for key, value := range dict {
		matched, _ := regexp.MatchString("scsi\\d+", key)

		if matched && !strings.Contains(value.(string), "cloudinit") {
			keySlice = append(keySlice, key)
		}
	}

	sort.Strings(keySlice)
	return keySlice
}

func diskNeedsUpdate(plannedDisk VmDisk, existingDisk VmDisk) bool {
	isEqual := plannedDisk.BusType.ValueString() == existingDisk.BusType.ValueString()
	isEqual = isEqual && plannedDisk.Order.ValueInt64() == existingDisk.Order.ValueInt64()
	isEqual = isEqual && plannedDisk.AsyncIo.ValueString() == existingDisk.AsyncIo.ValueString()
	isEqual = isEqual && plannedDisk.Size.ValueString() == existingDisk.Size.ValueString()
	isEqual = isEqual && plannedDisk.Cache.ValueString() == existingDisk.Cache.ValueString()
	isEqual = isEqual && plannedDisk.IoThread.ValueBool() == existingDisk.IoThread.ValueBool()
	isEqual = isEqual && plannedDisk.SsdEmulation.ValueBool() == existingDisk.SsdEmulation.ValueBool()
	isEqual = isEqual && plannedDisk.ReadOnly.ValueBool() == existingDisk.ReadOnly.ValueBool()
	isEqual = isEqual && plannedDisk.Replicate.ValueBool() == existingDisk.Replicate.ValueBool()
	isEqual = isEqual && plannedDisk.Discard.ValueBool() == existingDisk.Discard.ValueBool()
	isEqual = isEqual && plannedDisk.Backup.ValueBool() == existingDisk.Backup.ValueBool()
	isEqual = isEqual && plannedDisk.StorageLocation.ValueString() == existingDisk.StorageLocation.ValueString()
	return !isEqual
}

func mapPlannedDisksToExisting(plannedDisks []VmDisk, existingDisks []VmDisk) map[int]int {
	var diskMappings = make(map[int]int)
	for i, plannedDisk := range plannedDisks {
		fmt.Println(fmt.Sprintf("MAPPING DISK %s%d", plannedDisk.BusType.ValueString(), plannedDisk.Order.ValueInt64()))
		diskMappings[i] = -1
		for j, existingDisk := range existingDisks {
			if areDisksEqual(plannedDisk, existingDisk) {
				diskMappings[i] = j
				break
			}
		}
	}
	return diskMappings
}

func areDisksEqual(disk1 VmDisk, disk2 VmDisk) bool {
	isEqual := disk1.BusType.ValueString() == disk2.BusType.ValueString()
	isEqual = isEqual && disk1.Id.ValueInt64() == disk2.Id.ValueInt64()
	isEqual = isEqual && disk1.StorageLocation.ValueString() == disk2.StorageLocation.ValueString()
	return isEqual
}

func convertSizeToGibibytes(sizeString string) int64 {
	unitLabel := sizeString[len(sizeString)-1:]
	size, _ := strconv.ParseInt(sizeString[:len(sizeString)-1], 10, 64)
	switch unitLabel {
	case "T":
		return size * 1024
	case "M":
		return size / 1024
	case "P":
		return size * 1024 * 1024
	}

	return size
}