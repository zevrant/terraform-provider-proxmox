package vm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-proxmox/proxmox_client"
	"terraform-provider-proxmox/services"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type DiskService interface {
	//AssignDiskIds(vmModel proxmoxTypes.VmModel) proxmoxTypes.VmModel
	UpdateDisksFromQemuResponse(otherFields map[string]interface{}, vmModel *proxmoxTypes.VmModel, plan *proxmoxTypes.VmModel) []proxmoxTypes.VmDisk
	AttachVmDiskRequests(disks []proxmoxTypes.VmDisk, params *url.Values, vmId *string, cloudInitEnabled bool, createNew bool)
	GetDiskKeysFromJsonDict(dict map[string]interface{}) []string
	GetDiskFromState(state proxmoxTypes.VmModel, diskName string) proxmoxTypes.VmDisk
	MapPlannedDisksToExisting(plannedDisks []proxmoxTypes.VmDisk, existingDisks []proxmoxTypes.VmDisk) (map[int]int, []proxmoxTypes.VmDisk)
	FindDiskIndex(diskSlice []proxmoxTypes.VmDisk, toBeFound proxmoxTypes.VmDisk) int
	findDiskIndexHelper(diskSlice []proxmoxTypes.VmDisk, toBeFound proxmoxTypes.VmDisk, startIndex int, endIndex int) int
	AreTheseDisksTheSame(disk1 proxmoxTypes.VmDisk, disk2 proxmoxTypes.VmDisk) bool
	ResizeImportedDisks(vmIf *string, nodeName *string, disks []proxmoxTypes.VmDisk) error
	UpdateDisksWithUserValues(disks []proxmoxTypes.VmDisk, plan *proxmoxTypes.VmModel)
	CompareVmDisks(current *proxmoxTypes.VmModel, planned *proxmoxTypes.VmModel) ([]proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk)
	DeleteVmDisk(disk *proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
	AddVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
	ResizeDisk(disk *proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
	DeleteVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
	UpdateVmDisks(toBeUpdated []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
	ResizeVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error
}

type DiskServiceImpl struct {
	tfContext           context.Context
	proxmoxClient       proxmox_client.ProxmoxClient
	taskService         services.TaskService
	proxmoxUtilsService services.ProxmoxUtilService
}

func NewDiskService(tfCtx context.Context, client proxmox_client.ProxmoxClient, proxmoxUtils services.ProxmoxUtilService, taskService services.TaskService) DiskService {
	diskService := DiskServiceImpl{
		tfContext:           tfCtx,
		proxmoxClient:       client,
		proxmoxUtilsService: proxmoxUtils,
		taskService:         taskService,
	}

	return &diskService
}

func (diskService *DiskServiceImpl) UpdateDisksFromQemuResponse(otherFields map[string]interface{}, vmModel *proxmoxTypes.VmModel, plan *proxmoxTypes.VmModel) []proxmoxTypes.VmDisk {
	var keySlice = diskService.GetDiskKeysFromJsonDict(otherFields)
	var disks []proxmoxTypes.VmDisk
	for _, key := range keySlice {
		disk := otherFields[key].(string)

		diskParts := strings.Split(disk, ",")
		diskFieldMap := diskService.proxmoxUtilsService.MapKeyValuePairsToMap(diskParts[1:])
		cache := diskFieldMap["cache"]
		storageLocation := strings.Split(diskParts[0], ":")[0]
		if cache == "" {
			cache = "default"
		}

		diskNumber, _ := strconv.ParseInt(strings.Split(strings.Split(diskParts[0], ":")[1], "-")[3], 10, 64)
		order, _ := strconv.Atoi(key[4:len(key)])
		newVmDisk := proxmoxTypes.VmDisk{
			Id:              types.Int64Value(diskNumber),
			BusType:         types.StringValue(key[:4]),
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

		if newVmDisk.AsyncIo.ValueString() == "" {
			newVmDisk.AsyncIo = types.StringValue("default")
		}
		disks = append(disks, newVmDisk)
		sort.Slice(disks, func(i, j int) bool {
			return strings.Compare(diskService.GetDiskName(disks[i]), diskService.GetDiskName(disks[j])) < 0
		})
	}
	diskService.UpdateDisksWithUserValues(disks, plan)
	return disks
}

func (diskService *DiskServiceImpl) AttachVmDiskRequests(disks []proxmoxTypes.VmDisk, params *url.Values, vmId *string, cloudInitEnabled bool, createNew bool) {
	for _, disk := range disks {
		//local-zfs:vm-140-disk-0,aio=io_uring,backup=0,cache=directsync,discard=on,iothread=1,replicate=0,ro=1,size=32G,ssd=1
		var diskString string
		if createNew && disk.ImportFrom.ValueString() == "" {
			diskString = fmt.Sprintf("%s:%s,size=%s", disk.StorageLocation.ValueString(), disk.Size.ValueString()[:len(disk.Size.ValueString())-1], disk.Size.ValueString())
		} else if disk.ImportFrom.ValueString() == "" {
			diskString = fmt.Sprintf("%s:vm-%d-disk-%d,size=%s", disk.StorageLocation.ValueString(), vmId, disk.Id.ValueInt64(), disk.Size.ValueString())
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
		diskString = fmt.Sprintf("%s,iothread=%s", diskString, diskService.proxmoxUtilsService.MapBoolToProxmoxString(disk.IoThread.ValueBool()))
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

func (diskService *DiskServiceImpl) GetDiskKeysFromJsonDict(dict map[string]interface{}) []string {
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

func (diskService *DiskServiceImpl) GetDiskFromState(state proxmoxTypes.VmModel, diskName string) proxmoxTypes.VmDisk {
	for _, disk := range state.Disks {
		if diskService.GetDiskName(disk) == diskName {
			return disk
		}
	}
	return proxmoxTypes.VmDisk{}
}

func (diskService *DiskServiceImpl) GetDiskName(disk proxmoxTypes.VmDisk) string {
	return fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64())
}

func (diskService *DiskServiceImpl) MapPlannedDisksToExisting(plannedDisks []proxmoxTypes.VmDisk, existingDisks []proxmoxTypes.VmDisk) (map[int]int, []proxmoxTypes.VmDisk) {
	var diskMappings = make(map[int]int)
	var disksToBeRemoved []proxmoxTypes.VmDisk

	fmt.Println("existing disks")
	for _, disk := range existingDisks {
		fmt.Print(diskService.GetDiskName(disk) + "\n")
	}

	fmt.Println("planned disks")
	for _, disk := range plannedDisks {
		fmt.Print(diskService.GetDiskName(disk) + "\n")
	}

	for i, existingDisk := range existingDisks {
		//TODO: VERY UNDEEDED
		existingDiskIndex := diskService.FindDiskIndex(plannedDisks, existingDisk)

		if existingDiskIndex == -1 {
			fmt.Println(fmt.Sprintf("Existing disk %s not found in plan, marking for deletion", diskService.GetDiskName(existingDisk)))
			disksToBeRemoved = append(disksToBeRemoved, existingDisk)
		} else {
			fmt.Println(fmt.Sprintf("Mapping Disk disk %s to %d", diskService.GetDiskName(existingDisk), i))
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
		fmt.Print(diskService.GetDiskName(disk) + "\n")
	}

	return diskMappings, disksToBeRemoved
}

//func (diskService *DiskServiceImpl) removeDisks(disksToBeRemoved []VmDisk, diskSlice []VmDisk) []VmDisk {
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

func (diskService *DiskServiceImpl) FindDiskIndex(diskSlice []proxmoxTypes.VmDisk, toBeFound proxmoxTypes.VmDisk) int {
	fmt.Printf("finding %s with ID %d\n", diskService.GetDiskName(toBeFound), toBeFound.Id.ValueInt64())
	for _, disk := range diskSlice {
		fmt.Print(diskService.GetDiskName(disk) + " ")
	}
	index := diskService.findDiskIndexHelper(diskSlice, toBeFound, 0, len(diskSlice)-1)
	fmt.Printf("Found index is for %s is %d", diskService.GetDiskName(toBeFound), index)
	return index
}

func (diskService *DiskServiceImpl) findDiskIndexHelper(diskSlice []proxmoxTypes.VmDisk, toBeFound proxmoxTypes.VmDisk, startIndex int, endIndex int) int {

	if startIndex < 0 || len(diskSlice) == 0 {
		return -1
	} else if startIndex == endIndex {
		if diskService.AreTheseDisksTheSame(diskSlice[startIndex], toBeFound) {
			return startIndex
		} else {
			return -1
		}
	} else if diskService.AreTheseDisksTheSame(diskSlice[startIndex], toBeFound) {
		return startIndex
	} else if diskService.AreTheseDisksTheSame(diskSlice[endIndex], toBeFound) {
		return endIndex
	} else if endIndex < startIndex {
		return -1
	}

	indexDiff := endIndex - startIndex

	halfWayIndex := indexDiff / 2
	fmt.Printf("start %d, end %d, middle %d \n", startIndex, endIndex, halfWayIndex)
	halfWayComparison := strings.Compare(diskService.GetDiskName(diskSlice[halfWayIndex]), diskService.GetDiskName(toBeFound))
	if halfWayComparison > 0 {
		return diskService.findDiskIndexHelper(diskSlice, toBeFound, 0, halfWayIndex)
	} else if halfWayComparison < 0 {
		return diskService.findDiskIndexHelper(diskSlice, toBeFound, endIndex-halfWayIndex, endIndex)
	} else {
		return halfWayIndex
	}
}

func (diskService *DiskServiceImpl) AreTheseDisksTheSame(disk1 proxmoxTypes.VmDisk, disk2 proxmoxTypes.VmDisk) bool {
	fmt.Printf("busType %s %s\n", disk1.BusType.ValueString(), disk2.BusType.ValueString())
	fmt.Printf("Order %d %d\n", disk1.Order.ValueInt64(), disk2.Order.ValueInt64())
	isEqual := disk1.BusType.ValueString() == disk2.BusType.ValueString()
	isEqual = isEqual && disk1.Order.ValueInt64() == disk2.Order.ValueInt64()
	return isEqual
}

func (diskService *DiskServiceImpl) ResizeImportedDisks(vmId *string, nodeName *string, disks []proxmoxTypes.VmDisk) error {
	for _, disk := range disks {
		tflog.Info(diskService.tfContext, fmt.Sprintf("Import from is %s", disk.ImportFrom.ValueString()))

		if disk.ImportFrom.ValueString() != "" {
			params := url.Values{}
			tflog.Info(diskService.tfContext, "Resizing imported disk")

			params.Add("disk", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
			params.Add("size", disk.Size.ValueString())

			taskResponse, resizeDiskError := diskService.proxmoxClient.ResizeVmDisk(params, nodeName, vmId)

			if resizeDiskError != nil {
				return errors.New(fmt.Sprintf("Failed to resize imported disk %s", resizeDiskError.Error()))
			}

			taskCompletionError := diskService.taskService.WaitForTaskCompletion(nodeName, taskResponse)

			if taskCompletionError != nil {
				return errors.New(fmt.Sprintf("Failed to wait for resize task completion %s", taskCompletionError.Error()))
			}
		}
	}
	return nil
}

func (diskService *DiskServiceImpl) UpdateDisksWithUserValues(disks []proxmoxTypes.VmDisk, plan *proxmoxTypes.VmModel) {
	for i, _ := range disks {
		if (disks[i].ImportFrom.IsNull() || disks[i].ImportFrom.ValueString() == "") && plan.Disks != nil && i < len(plan.Disks) && !plan.Disks[i].ImportFrom.IsNull() {
			disks[i].ImportFrom = plan.Disks[i].ImportFrom
			disks[i].Path = plan.Disks[i].Path
		}
	}
}

func (diskService *DiskServiceImpl) CompareVmDisks(existing *proxmoxTypes.VmModel, planned *proxmoxTypes.VmModel) ([]proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk, []proxmoxTypes.VmDisk) {
	var toBeAdded []proxmoxTypes.VmDisk
	var toBeRemoved []proxmoxTypes.VmDisk
	var toBeUpdated []proxmoxTypes.VmDisk
	var toBeResized []proxmoxTypes.VmDisk

	for i, _ := range existing.Disks {
		plannedIndex := -1
		if i < len(planned.Disks) && diskService.AreTheseDisksTheSame(existing.Disks[i], planned.Disks[i]) {
			plannedIndex = i
		}
		if plannedIndex == -1 {
			plannedIndex = diskService.FindDiskIndex(planned.Disks, existing.Disks[i])
		}

		if plannedIndex == -1 {
			toBeRemoved = append(toBeRemoved, existing.Disks[i])
			continue
		}
		existingDisk := existing.Disks[i]
		plannedDisk := planned.Disks[plannedIndex]

		isEqual := existingDisk.ImportFrom.ValueString() == plannedDisk.ImportFrom.ValueString()
		isEqual = isEqual && existingDisk.Path.ValueString() == plannedDisk.Path.ValueString()
		isEqual = isEqual && existingDisk.StorageLocation.ValueString() == plannedDisk.StorageLocation.ValueString()
		isEqual = isEqual && existingDisk.AsyncIo.ValueString() == plannedDisk.AsyncIo.ValueString()
		isEqual = isEqual && existingDisk.BusType.ValueString() == plannedDisk.BusType.ValueString()
		isEqual = isEqual && existingDisk.IoThread.ValueBool() == plannedDisk.IoThread.ValueBool()
		isEqual = isEqual && existingDisk.Cache.ValueString() == plannedDisk.Cache.ValueString()
		isEqual = isEqual && existingDisk.Replicate.ValueBool() == plannedDisk.Replicate.ValueBool()
		isEqual = isEqual && existingDisk.SsdEmulation.ValueBool() == plannedDisk.SsdEmulation.ValueBool()
		isEqual = isEqual && existingDisk.Backup.ValueBool() == plannedDisk.Backup.ValueBool()
		isEqual = isEqual && existingDisk.Order.ValueInt64() == plannedDisk.Order.ValueInt64()
		isEqual = isEqual && existingDisk.Discard.ValueBool() == plannedDisk.Discard.ValueBool()
		isEqual = isEqual && existingDisk.ReadOnly.ValueBool() == plannedDisk.ReadOnly.ValueBool()

		if !isEqual {
			toBeUpdated = append(toBeUpdated, planned.Disks[i])
		}

		if existingDisk.Size.ValueString() != plannedDisk.Size.ValueString() || existingDisk.Path.ValueString() != plannedDisk.Path.ValueString() {
			toBeResized = append(toBeResized, planned.Disks[i])
		}

	}
	for i, _ := range planned.Disks {
		if diskService.FindDiskIndex(existing.Disks, planned.Disks[i]) == -1 {
			toBeAdded = append(toBeAdded, planned.Disks[i])
		}
	}
	return toBeAdded, toBeUpdated, toBeRemoved, toBeResized
}

func (diskService *DiskServiceImpl) DeleteVmDisk(disk *proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {
	params := url.Values{}
	params.Add("delete", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))

	upid, updateVmErrror := diskService.proxmoxClient.UpdateVm(params, nodeName, vmId)

	if updateVmErrror != nil {
		return updateVmErrror
	}

	waitForTaskCompletionError := diskService.taskService.WaitForTaskCompletion(nodeName, upid)

	if waitForTaskCompletionError != nil {
		return waitForTaskCompletionError
	}

	params.Set("delete", "unused0")

	upid, updateVmErrror = diskService.proxmoxClient.UpdateVm(params, nodeName, vmId)

	if updateVmErrror != nil {
		return updateVmErrror
	}

	waitForTaskCompletionError = diskService.taskService.WaitForTaskCompletion(nodeName, upid)

	if waitForTaskCompletionError != nil {
		return waitForTaskCompletionError
	}

	return nil
}

func (diskService *DiskServiceImpl) AddVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {

	if len(disks) == 0 {
		return nil
	}

	params := url.Values{}

	diskService.AttachVmDiskRequests(disks, &params, vmId, false, true)

	upid, vmUpdateError := diskService.proxmoxClient.UpdateVm(params, nodeName, vmId)

	if vmUpdateError != nil {
		return vmUpdateError
	}

	taskCompletionError := diskService.taskService.WaitForTaskCompletion(nodeName, upid)

	if taskCompletionError != nil {
		return taskCompletionError
	}

	for _, disk := range disks {
		if disk.ImportFrom.ValueString() != "" {
			resizeDiskError := diskService.ResizeDisk(&disk, nodeName, vmId)
			if resizeDiskError != nil {
				return resizeDiskError
			}

		}
	}
	return nil
}

func (diskService *DiskServiceImpl) ResizeDisk(disk *proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {
	params := url.Values{}
	tflog.Info(diskService.tfContext, "Resizing imported disk")

	params.Add("disk", fmt.Sprintf("%s%d", disk.BusType.ValueString(), disk.Order.ValueInt64()))
	params.Add("size", disk.Size.ValueString())

	taskResponse, resizeDiskError := diskService.proxmoxClient.ResizeVmDisk(params, nodeName, vmId)

	if resizeDiskError != nil {
		return resizeDiskError
	}

	taskCompletionError := diskService.taskService.WaitForTaskCompletion(nodeName, taskResponse)

	if taskCompletionError != nil {
		return taskCompletionError
	}
	return nil
}

func (diskService *DiskServiceImpl) DeleteVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {
	for _, disk := range disks {
		diskDeletionError := diskService.DeleteVmDisk(&disk, nodeName, vmId)

		if diskDeletionError != nil {

			return diskDeletionError
		}
	}
	return nil
}

func (diskService *DiskServiceImpl) UpdateVmDisks(toBeUpdated []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {
	if len(toBeUpdated) == 0 {
		return nil
	}
	params := url.Values{}
	diskService.AttachVmDiskRequests(toBeUpdated, &params, vmId, false, true)
	upid, vmUpdateError := diskService.proxmoxClient.UpdateVm(params, nodeName, vmId)

	if vmUpdateError != nil {
		return vmUpdateError
	}

	taskCompletionError := diskService.taskService.WaitForTaskCompletion(nodeName, upid)

	if taskCompletionError != nil {
		return taskCompletionError
	}

	return nil
}

func (diskService *DiskServiceImpl) ResizeVmDisks(disks []proxmoxTypes.VmDisk, nodeName *string, vmId *string) error {
	for _, disk := range disks {
		resizeDiskError := diskService.ResizeDisk(&disk, nodeName, vmId)
		if resizeDiskError != nil {
			return resizeDiskError
		}
	}
	return nil
}
