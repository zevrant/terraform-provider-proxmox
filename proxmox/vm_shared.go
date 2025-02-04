package proxmox

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func updateVmModelFromResponse(vmModel VmModel, response proxmox_client.QemuResponse, tfContext *context.Context) VmModel {
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
	unescapedSshKeys, _ := url.PathUnescape(response.Data.SshKeys)
	vmModel.SshKeys, _ = types.ListValueFrom(*tfContext, types.StringType, strings.Split(unescapedSshKeys, "\\n"))
	startupOrder, _ := strconv.ParseInt(strings.Replace(response.Data.HostStartupOrder, "order=", "", 1), 10, 64)
	vmModel.DefaultUser = types.StringValue(response.Data.CiUser)
	vmModel.HostStartupOrder = types.Int64Value(startupOrder)
	if response.Data.Bios == "" {
		vmModel.Bios = types.StringValue("seabios")
	} else {
		vmModel.Bios = types.StringValue(response.Data.Bios)
	}
	vmModel.Disks = updateDisksFromQemuResponse(response.Data.OtherFields, vmModel)
	vmModel.NetworkInterfaces = mapNetworkInterfacesFromQemuResponse(response.Data.OtherFields)
	vmModel.IpConfigurations = mapIpConfigsFromQemuResponse(response.Data.OtherFields)

	return vmModel
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

func assignDiskIds(vmModel VmModel) VmModel {
	vmDisks := vmModel.Disks

	storageLocations := make([]string, 0)

	for _, disk := range vmDisks {
		if !slices.Contains(storageLocations, disk.StorageLocation.ValueString()) {
			storageLocations = append(storageLocations, disk.StorageLocation.ValueString())
		}
	}

	storageIdMapping := make(map[string]int64)

	for _, location := range storageLocations {
		storageIdMapping[location] = int64(0)
	}

	newDisks := make([]VmDisk, 0)
	for _, disk := range vmDisks {
		currentId := storageIdMapping[disk.StorageLocation.ValueString()]
		storageIdMapping[disk.StorageLocation.ValueString()] = currentId + 1
		disk.Id = types.Int64Value(currentId)
		newDisks = append(newDisks, disk)
	}
	vmModel.Disks = newDisks
	return vmModel
}

func updateDisksFromQemuResponse(otherFields map[string]interface{}, vmModel VmModel) []VmDisk {
	var keySlice = getDiskKeysFromJsonDict(otherFields)
	var disks []VmDisk
	for order, key := range keySlice {
		disk := otherFields[key].(string)

		diskParts := strings.Split(disk, ",")
		diskFieldMap := mapKeyValuePairsToMap(diskParts[1:])
		cache := diskFieldMap["cache"]
		storageLocation := strings.Split(diskParts[0], ":")[0]
		if cache == "" {
			cache = "default"
		}

		diskNumber, _ := strconv.ParseInt(strings.Split(strings.Split(diskParts[0], ":")[1], "-")[3], 10, 64)
		if strings.Contains(diskParts[0], "cloudinit") {
			//e.g. local-zfs:vm-1040-cloudinit
			vmModel.CloudInitStorageName = types.StringValue(strings.Split(diskParts[0], ":")[0])
			continue
		}

		//for _, plannedDisk := range vmModel.Disks {
		//	if plannedDisk.BusType.ValueString() == key[:len("scsi")] && plannedDisk.Order.ValueInt64() == int64(order) && plannedDisk.StorageLocation.ValueString() == storageLocation {
		newVmDisk := VmDisk{
			Id:              types.Int64Value(diskNumber),
			BusType:         types.StringValue(key[:len("scsi")]),
			StorageLocation: types.StringValue(storageLocation),
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
			ImportFrom:      types.StringValue(""),
			Path:            types.StringValue(""),
		}

		plannedDiskIndex := findDiskIndex(vmModel.Disks, newVmDisk)
		if plannedDiskIndex != -1 {
			newVmDisk.ImportFrom = vmModel.Disks[plannedDiskIndex].ImportFrom
			newVmDisk.Path = vmModel.Disks[plannedDiskIndex].Path
		}

		if newVmDisk.AsyncIo.ValueString() == "" {
			newVmDisk.AsyncIo = types.StringValue("default")
		}
		disks = append(disks, newVmDisk)
		sort.Slice(disks, func(i, j int) bool {
			return strings.Compare(getDiskName(disks[i]), getDiskName(disks[j])) < 0
		})
		//}
		//}
	}
	return disks
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

		if mappedNicFields["mtu"] != "" {
			mtu, _ := strconv.ParseInt(mappedNicFields["mtu"], 10, 64)
			newVmNic.Mtu = types.Int64Value(mtu)
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

func createVmRequest(vmModel VmModel, tfContext *context.Context, cloudInitEnabled bool, createNew bool) url.Values {
	params := url.Values{}
	params.Add("vmid", vmModel.VmId.ValueString())
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
	_ = vmModel.SshKeys.ElementsAs(*tfContext, &keysList, false)

	for _, key := range keysList {
		if sshKeys == "" {
			sshKeys = key.ValueString()
		} else {
			sshKeys += fmt.Sprintf("\n%s", key.ValueString())
		}
	}

	onBoot := "0"

	if vmModel.OnBoot.ValueBool() {
		onBoot = "1"
	}

	params.Add("acpi", mapBoolToProxmoxString(vmModel.Acpi.ValueBool()))
	params.Add("agent", mapBoolToProxmoxString(vmModel.Agent.ValueBool()))
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
		params.Add("sshkeys", strings.ReplaceAll(strings.ReplaceAll(url.PathEscape(sshKeys), "+", "%2B"), "=", "%3D"))
	}
	params.Add("cores", vmModel.Cores.String())
	params.Add("tags", tags)
	params.Add("startup", fmt.Sprintf("order=%d", vmModel.HostStartupOrder.ValueInt64()))
	params.Add("protection", mapBoolToProxmoxString(vmModel.Protection.ValueBool()))
	params.Add("ostype", vmModel.OsType.ValueString())
	params.Add("onboot", onBoot)
	if vmModel.DefaultUser.ValueString() != "" {
		params.Add("ciuser", vmModel.DefaultUser.ValueString())
	}

	if createNew {
		attachVmDiskRequests(vmModel.Disks, &params, vmModel.VmId.ValueString(), cloudInitEnabled, createNew)
	}
	attachVmNicRequests(vmModel, &params)
	return params
}

func attachVmDiskRequests(disks []VmDisk, params *url.Values, vmId string, cloudInitEnabled bool, createNew bool) {
	for _, disk := range disks {
		//local-zfs:vm-140-disk-0,aio=io_uring,backup=0,cache=directsync,discard=on,iothread=1,replicate=0,ro=1,size=32G,ssd=1
		var diskString string
		if createNew && disk.ImportFrom.ValueString() == "" {
			diskString = fmt.Sprintf("%s:%s,size=%s", disk.StorageLocation.ValueString(), disk.Size.ValueString()[:len(disk.Size.ValueString())-1], disk.Size.ValueString())
		} else if disk.ImportFrom.ValueString() == "" {
			diskString = fmt.Sprintf("%s:vm-%s-disk-%d,size=%s", disk.StorageLocation.ValueString(), vmId, disk.Id.ValueInt64(), disk.Size.ValueString())
		} else if createNew && disk.ImportFrom.ValueString() != "" {
			//size is ignored so we don't include it here
			//for information on why it's this way see https://www.reddit.com/r/Proxmox/comments/y51x5h/comment/jujh7zi/?utm_source=share&utm_medium=web3x&utm_name=web3xcss&utm_term=1&utm_content=share_button
			diskString = fmt.Sprintf("%s:0,import-from=%s:%s", disk.StorageLocation.ValueString(), disk.ImportFrom.ValueString(), disk.Path.ValueString())
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
		mtu := ""
		if !nicConfig.Mtu.IsNull() {
			mtu = fmt.Sprintf(",mtu=%d", nicConfig.Mtu.ValueInt64())
		}
		params.Add(fmt.Sprintf("net%d", nicConfig.Order.ValueInt64()), fmt.Sprintf("%s=%s,bridge=%s,firewall=%s%s", nicConfig.Type.ValueString(), nicConfig.MacAddress.ValueString(), nicConfig.Bridge.ValueString(), mapBoolToProxmoxString(nicConfig.Firewall.ValueBool()), mtu))
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

func getDiskFromState(state VmModel, diskName string) VmDisk {
	for _, disk := range state.Disks {
		if getDiskName(disk) == diskName {
			return disk
		}
	}
	return VmDisk{}
}

func getDiskName(disk VmDisk) string {
	return fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64())
}

func mapPlannedDisksToExisting(plannedDisks []VmDisk, existingDisks []VmDisk) (map[int]int, []VmDisk) {
	var diskMappings = make(map[int]int)
	var disksToBeRemoved []VmDisk

	fmt.Println("existing disks")
	for _, disk := range existingDisks {
		fmt.Print(getDiskName(disk) + "\n")
	}

	fmt.Println("planned disks")
	for _, disk := range plannedDisks {
		fmt.Print(getDiskName(disk) + "\n")
	}

	for i, existingDisk := range existingDisks {
		existingDiskIndex := findDiskIndex(plannedDisks, existingDisk)

		if existingDiskIndex == -1 {
			fmt.Println(fmt.Sprintf("Existing disk %s not found in plan, marking for deletion", getDiskName(existingDisk)))
			disksToBeRemoved = append(disksToBeRemoved, existingDisk)
		} else {
			fmt.Println(fmt.Sprintf("Mapping Disk disk %s to %d", getDiskName(existingDisk), i))
			diskMappings[existingDiskIndex] = i
		}
	}
	for i, _ := range plannedDisks {
		_, exists := diskMappings[i]

		if !exists {
			diskMappings[i] = -1
		}
	}

	fmt.Printf("disks to remove %d \n", len(disksToBeRemoved))
	for _, disk := range disksToBeRemoved {
		fmt.Print(getDiskName(disk) + "\n")
	}

	return diskMappings, disksToBeRemoved
}

//func removeDisks(disksToBeRemoved []VmDisk, diskSlice []VmDisk) []VmDisk {
//	localDiskSlice := diskSlice
//	sort.Slice(diskSlice, func(i, j int) bool {
//		return strings.Compare(getDiskName(localDiskSlice[i]), getDiskName(localDiskSlice[j])) < 0
//	})
//	for _, toRemove := range disksToBeRemoved {
//		index := findDiskIndex(diskSlice, toRemove)
//		if index != 0 && index != len(diskSlice)-1 {
//			localDiskSlice = slices.Concat(localDiskSlice[0:index], disksToBeRemoved[index+1:]) //remove from middle
//		} else if index == 0 {
//			localDiskSlice = localDiskSlice[1:]
//		} else { //index is the end of the list
//			localDiskSlice = localDiskSlice[0 : index-1]
//		}
//	}
//}

func findDiskIndex(diskSlice []VmDisk, toBeFound VmDisk) int {
	fmt.Printf("finding %s with ID %d\n", getDiskName(toBeFound), toBeFound.Id.ValueInt64())
	for _, disk := range diskSlice {
		fmt.Print(getDiskName(disk) + " ")
	}
	index := findDiskIndexHelper(diskSlice, toBeFound, 0, len(diskSlice)-1)
	fmt.Printf("Found index is for %s is %d", getDiskName(toBeFound), index)
	return index
}

func findDiskIndexHelper(diskSlice []VmDisk, toBeFound VmDisk, startIndex int, endIndex int) int {

	if startIndex < 0 || len(diskSlice) == 0 {
		return -1
	} else if startIndex == endIndex {
		if areDisksEqual(diskSlice[startIndex], toBeFound) {
			return startIndex
		} else {
			return -1
		}
	} else if areDisksEqual(diskSlice[startIndex], toBeFound) {
		return startIndex
	} else if areDisksEqual(diskSlice[endIndex], toBeFound) {
		return endIndex
	} else if endIndex < startIndex {
		return -1
	}

	indexDiff := endIndex - startIndex

	halfWayIndex := indexDiff / 2
	fmt.Printf("start %d, end %d, middle %d \n", startIndex, endIndex, halfWayIndex)
	halfWayComparison := strings.Compare(getDiskName(diskSlice[halfWayIndex]), getDiskName(toBeFound))
	if halfWayComparison > 0 {
		return findDiskIndexHelper(diskSlice, toBeFound, 0, halfWayIndex)
	} else if halfWayComparison < 0 {
		return findDiskIndexHelper(diskSlice, toBeFound, endIndex-halfWayIndex, endIndex)
	} else {
		return halfWayIndex
	}
}

func areDisksEqual(disk1 VmDisk, disk2 VmDisk) bool {
	fmt.Printf("busType %s %s\n", disk1.BusType.ValueString(), disk2.BusType.ValueString())
	fmt.Printf("ID %d %d\n", disk1.Order.ValueInt64(), disk2.Order.ValueInt64())
	fmt.Printf("StorageLocation %s %s\n", disk1.StorageLocation.ValueString(), disk2.StorageLocation.ValueString())
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

func shutdownVm(nodeName string, vmId string, diags diag.Diagnostics, client *proxmox_client.Client) error {
	shutdownUpid, shutdownVmError := client.ShutdownVm(nodeName, vmId)
	if shutdownVmError != nil {
		diags.AddError("Failed to shutdown VM", shutdownVmError.Error())
		return shutdownVmError
	}
	waitForShutdownError := waitForTaskCompletion(nodeName, shutdownUpid, client)

	if waitForShutdownError != nil {
		diags.AddError("Failed to shutdown VM", waitForShutdownError.Error())
		return waitForShutdownError
	}

	vmStatus, getStatusError := client.GetVmStatus(nodeName, vmId)
	if getStatusError != nil {
		diags.AddError("Failed to get vm status after shutdown event", getStatusError.Error())
		return getStatusError
	}

	if vmStatus != "stopped" {
		diags.AddError("Vm shutdown did not result in the vm shutting down.", fmt.Sprintf("Expected status stopped, got %s", vmStatus))
		return errors.New("unexpected post shutdown state")
	}
	return nil
}

func startVm(nodeName string, vmId string, diags diag.Diagnostics, client *proxmox_client.Client) error {
	shutdownUpid, startVmError := client.StartVm(nodeName, vmId)
	if startVmError != nil {
		diags.AddError("Failed to start VM", startVmError.Error())
		return startVmError
	}
	waitForStartupError := waitForTaskCompletion(nodeName, shutdownUpid, client)

	if waitForStartupError != nil {
		diags.AddError("Failed to start VM", waitForStartupError.Error())
		return waitForStartupError
	}

	vmStatus, getStatusError := client.GetVmStatus(nodeName, vmId)
	if getStatusError != nil {
		diags.AddError("Failed to get vm status after startup", getStatusError.Error())
		return getStatusError
	}

	if vmStatus != "running" {
		diags.AddError("Vm startup did leave the vm in a running state.", fmt.Sprintf("Expected status running, got %s", vmStatus))
		return errors.New("unexpected post shutdown state")
	}
	return nil
}
