package vm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"terraform-provider-proxmox/proxmox_client"
	"terraform-provider-proxmox/services"
	proxmoxTypes "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VmService interface {
	UpdateVmModelFromResponse(vmModel *proxmoxTypes.VmModel, plan *proxmoxTypes.VmModel, response *proxmoxTypes.QemuResponse) *proxmoxTypes.VmModel
	MapNetworkInterfacesFromQemuResponse(otherFields map[string]interface{}) []proxmoxTypes.VmNetworkInterface
	CreateVmRequest(vmModel *proxmoxTypes.VmModel, cloudInitEnabled bool, createNew bool) url.Values
	ShutdownVm(nodeName *string, vmId *string) error
	StartVm(nodeName *string, vmId *string) error
	MapIpConfigsFromQemuResponse(otherFields map[string]interface{}) []proxmoxTypes.VmIpConfig
	AttachVmNicRequests(vmModel *proxmoxTypes.VmModel, params *url.Values)
	FindVmByNodeWithId(nodeName *string, vmId *string) (*proxmoxTypes.QemuResponse, error)
	SearchVmById(vmId *string) (*proxmoxTypes.QemuResponse, *string, error)
	GetVm(nodeName *string, vmId *string) (*proxmoxTypes.QemuResponse, *string, error)
	UpdatePowerState(model *proxmoxTypes.VmModel) error
	CreateVm(plan *proxmoxTypes.VmModel) error
	MatchVmPowerState(plan *proxmoxTypes.VmModel, currentState *proxmoxTypes.VmModel) error
	DeleteVm(nodeName *string, vmId *string) error
	UpdateVm(plan *proxmoxTypes.VmModel, nodeName *string, vmId *string) error
	MigrateVm(currentNode *string, newNode *string, vmId *string) error
}

type VmServiceImpl struct {
	tfContext     context.Context
	proxmoxClient proxmox_client.ProxmoxClient
	diskService   DiskService
	proxmoxUtils  services.ProxmoxUtilService
	taskService   services.TaskService
}

func NewVmService(ctx context.Context, proxmoxClient proxmox_client.ProxmoxClient, diskService DiskService, proxmoxUtils services.ProxmoxUtilService, taskService services.TaskService) VmService {
	vmService := VmServiceImpl{
		tfContext:     ctx,
		proxmoxClient: proxmoxClient,
		diskService:   diskService,
		proxmoxUtils:  proxmoxUtils,
		taskService:   taskService,
	}

	return &vmService
}

func (vmService *VmServiceImpl) UpdateVmModelFromResponse(vmModel *proxmoxTypes.VmModel, plan *proxmoxTypes.VmModel, response *proxmoxTypes.QemuResponse) *proxmoxTypes.VmModel {
	memory, _ := strconv.ParseInt(response.Data.Memory, 10, 64)
	tags := strings.Split(strings.Trim(response.Data.Tags, " "), ";")

	if response.Data.Tags == " " {
		tags = []string{}
	}

	vmModel.Cpu = types.StringValue(response.Data.Cpu)
	tflog.Debug(vmService.tfContext, fmt.Sprintf("Setting cpu type to %s", response.Data.Cpu))
	vmModel.Memory = types.Int64Value(memory)
	vmModel.Tags, _ = types.ListValueFrom(vmService.tfContext, types.StringType, tags)
	vmModel.Name = types.StringValue(response.Data.Name)
	vmModel.OnBoot = types.BoolValue(response.Data.OnBoot == 1)
	vmModel.Description = types.StringValue(response.Data.Description)
	vmModel.VmGenId = types.StringValue(response.Data.VmGenId)
	vmModel.Sockets = types.Int64Value(int64(response.Data.Sockets))
	vmModel.OsType = types.StringValue(response.Data.OsType)
	vmModel.ScsiHw = types.StringValue(response.Data.ScsiHw)
	vmModel.Agent = types.BoolValue(response.Data.Agent == "1") //No clue why this is coming back a string as opposed to an int like the others
	vmModel.BootOrder, _ = types.ListValueFrom(vmService.tfContext, types.StringType, strings.Split(strings.Replace(response.Data.Boot, "order=", "", 1), ","))
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
	vmModel.SshKeys, _ = types.ListValueFrom(vmService.tfContext, types.StringType, strings.Split(unescapedSshKeys, "\\n"))
	startupOrder, _ := strconv.ParseInt(strings.Replace(response.Data.HostStartupOrder, "order=", "", 1), 10, 64)
	vmModel.DefaultUser = types.StringValue(response.Data.CiUser)
	vmModel.HostStartupOrder = types.Int64Value(startupOrder)
	if response.Data.Bios == "" {
		vmModel.Bios = types.StringValue("seabios")
	} else {
		vmModel.Bios = types.StringValue(response.Data.Bios)
	}
	vmModel.Disks = vmService.diskService.UpdateDisksFromQemuResponse(response.Data.OtherFields, vmModel, plan)
	vmModel.NetworkInterfaces = vmService.MapNetworkInterfacesFromQemuResponse(response.Data.OtherFields)
	vmModel.IpConfigurations = vmService.MapIpConfigsFromQemuResponse(response.Data.OtherFields)

	return vmModel
}

func (vmService *VmServiceImpl) MapNetworkInterfacesFromQemuResponse(otherFields map[string]interface{}) []proxmoxTypes.VmNetworkInterface {
	var vmNics []proxmoxTypes.VmNetworkInterface
	var keySlice []string
	for key, _ := range otherFields {
		matched, _ := regexp.MatchString("net\\d+", key)
		if matched {
			keySlice = append(keySlice, key)
		}
	}

	sort.Strings(keySlice)
	var networkInterfaceTypes []string = strings.Split(proxmoxTypes.NetworkInterfaceTypes, " | ")
	for order, key := range keySlice {
		if !strings.Contains(key, "net") {
			continue
		}

		nic := otherFields[key].(string)

		nicParts := strings.Split(nic, ",")

		var networkInterfaceType = ""

		mappedNicFields := vmService.proxmoxUtils.MapKeyValuePairsToMap(nicParts)

		for netInterfaceKey, _ := range mappedNicFields {
			if slices.Contains(networkInterfaceTypes, netInterfaceKey) {
				networkInterfaceType = netInterfaceKey
			}
		}

		newVmNic := proxmoxTypes.VmNetworkInterface{
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

func (vmService *VmServiceImpl) CreateVmRequest(vmModel *proxmoxTypes.VmModel, cloudInitEnabled bool, createNew bool) url.Values {
	params := url.Values{}
	params.Add("vmid", vmModel.VmId.ValueString())
	params.Add("name", vmModel.Name.ValueString())

	tagsList := make([]types.String, 0, len(vmModel.Tags.Elements()))
	_ = vmModel.Tags.ElementsAs(vmService.tfContext, &tagsList, false)
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
	_ = vmModel.BootOrder.ElementsAs(vmService.tfContext, &disks, false)
	for _, disk := range disks {
		if bootOrder == "" {
			bootOrder = disk.ValueString()
		} else {
			bootOrder += ";" + disk.ValueString()
		}
	}

	sshKeys := ""
	keysList := make([]types.String, 0, len(vmModel.SshKeys.Elements()))
	_ = vmModel.SshKeys.ElementsAs(vmService.tfContext, &keysList, false)

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

	params.Add("acpi", vmService.proxmoxUtils.MapBoolToProxmoxString(vmModel.Acpi.ValueBool()))
	params.Add("agent", vmService.proxmoxUtils.MapBoolToProxmoxString(vmModel.Agent.ValueBool()))
	params.Add("bios", vmModel.Bios.ValueString())
	params.Add("boot", fmt.Sprintf("order=%s", bootOrder))
	params.Add("ciupgrade", vmService.proxmoxUtils.MapBoolToProxmoxString(vmModel.CloudInitUpgrade.ValueBool()))
	params.Add("cpu", vmModel.Cpu.ValueString())
	params.Add("hotplug", "network,usb")
	params.Add("cpulimit", vmModel.CpuLimit.String())
	params.Add("description", vmModel.Description.ValueString())
	for index, ipConfig := range vmModel.IpConfigurations {
		params.Add(fmt.Sprintf("ipconfig%d", index), fmt.Sprintf("gw=%s,ip=%s", ipConfig.Gateway.ValueString(), ipConfig.IpAddress.ValueString()))
	}
	params.Add("kvm", vmService.proxmoxUtils.MapBoolToProxmoxString(vmModel.Kvm.ValueBool()))
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
	params.Add("protection", vmService.proxmoxUtils.MapBoolToProxmoxString(vmModel.Protection.ValueBool()))
	params.Add("ostype", vmModel.OsType.ValueString())
	params.Add("onboot", onBoot)
	if vmModel.DefaultUser.ValueString() != "" {
		params.Add("ciuser", vmModel.DefaultUser.ValueString())
	}

	if createNew {
		vmService.diskService.AttachVmDiskRequests(vmModel.Disks, &params, vmModel.VmId.ValueStringPointer(), cloudInitEnabled, createNew)
	}
	vmService.AttachVmNicRequests(vmModel, &params)
	return params
}

func (vmService *VmServiceImpl) ShutdownVm(nodeName *string, vmId *string) error {
	shutdownUpid, shutdownVmError := vmService.proxmoxClient.ShutdownVm(nodeName, vmId)
	if shutdownVmError != nil {
		return shutdownVmError
	}
	waitForShutdownError := vmService.taskService.WaitForTaskCompletion(nodeName, shutdownUpid)

	if waitForShutdownError != nil {
		return waitForShutdownError
	}

	vmStatus, getStatusError := vmService.proxmoxClient.GetVmStatus(nodeName, vmId)
	if getStatusError != nil {
		return getStatusError
	}

	if vmStatus != "stopped" {
		return errors.New("unexpected post shutdown state")
	}
	return nil
}

func (vmService *VmServiceImpl) StartVm(nodeName *string, vmId *string) error {
	shutdownUpid, startVmError := vmService.proxmoxClient.StartVm(nodeName, vmId)
	if startVmError != nil {
		return errors.New(fmt.Sprintf("Failed to start VM: %s", startVmError.Error()))
	}
	waitForStartupError := vmService.taskService.WaitForTaskCompletion(nodeName, shutdownUpid)

	if waitForStartupError != nil {
		return waitForStartupError
	}

	vmStatus, getStatusError := vmService.proxmoxClient.GetVmStatus(nodeName, vmId)
	if getStatusError != nil {
		return getStatusError
	}

	if vmStatus != "running" {
		return errors.New("unexpected post shutdown state")
	}
	return nil
}

func (vmService *VmServiceImpl) MapIpConfigsFromQemuResponse(otherFields map[string]interface{}) []proxmoxTypes.VmIpConfig {
	var vmIpConfigs []proxmoxTypes.VmIpConfig
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

		mappedIpConfigFields := vmService.proxmoxUtils.MapKeyValuePairsToMap(nicParts)

		newVmNic := proxmoxTypes.VmIpConfig{
			IpAddress: types.StringValue(mappedIpConfigFields["ip"]),
			Gateway:   types.StringValue(mappedIpConfigFields["gw"]),
			Order:     types.Int64Value(int64(order)),
		}

		vmIpConfigs = append(vmIpConfigs, newVmNic)
	}
	return vmIpConfigs
}

func (vmService *VmServiceImpl) AttachVmNicRequests(vmModel *proxmoxTypes.VmModel, params *url.Values) {
	for _, nicConfig := range vmModel.NetworkInterfaces {
		mtu := ""
		if !nicConfig.Mtu.IsNull() {
			mtu = fmt.Sprintf(",mtu=%d", nicConfig.Mtu.ValueInt64())
		}
		params.Add(fmt.Sprintf("net%d", nicConfig.Order.ValueInt64()), fmt.Sprintf("%s=%s,bridge=%s,firewall=%s%s", nicConfig.Type.ValueString(), nicConfig.MacAddress.ValueString(), nicConfig.Bridge.ValueString(), vmService.proxmoxUtils.MapBoolToProxmoxString(nicConfig.Firewall.ValueBool()), mtu))
	}
}

func (vmService *VmServiceImpl) FindVmByNodeWithId(nodeName *string, vmId *string) (*proxmoxTypes.QemuResponse, error) {
	if nodeName == nil {
		return nil, errors.New("cannot retrieve vm by node without a valid node name, use searchVmById instead")
	}
	if vmId == nil {
		return nil, errors.New("cannot retrieve vm by id without a valid vm id")
	}

	vmResponse, searchVmError := vmService.proxmoxClient.GetVmById(nodeName, vmId)

	if searchVmError != nil {
		return nil, searchVmError
	}

	return vmResponse, nil
}

func (vmService *VmServiceImpl) SearchVmById(vmId *string) (*proxmoxTypes.QemuResponse, *string, error) {
	if vmId == nil {
		return nil, nil, errors.New("cannot search for vm by id without a valid vm id")
	}

	nodeList, listNodesError := vmService.proxmoxClient.ListNodes()

	if listNodesError != nil {
		return nil, nil, listNodesError
	}

	var vmResponse *proxmoxTypes.QemuResponse
	var searchVmError error
	var nodeName string
	for _, node := range nodeList.Data {

		tflog.Debug(vmService.tfContext, fmt.Sprintf("Node name is %s", node.Node))

		vmResponse, searchVmError = vmService.proxmoxClient.GetVmById(&node.Node, vmId)

		if searchVmError != nil && !strings.Contains(searchVmError.Error(), fmt.Sprintf("500 Configuration file 'nodes/%s/qemu-server/%s.conf' does not exist", node.Node, *vmId)) {
			return nil, nil, errors.New(fmt.Sprintf("Failed to search for node that VM lives on: %s", searchVmError.Error()))
		}
		if searchVmError == nil {
			nodeName = node.Node
			break
		}

	}

	if vmResponse == nil {
		return nil, nil, errors.New(fmt.Sprintf("Could not find vm for id %s within the cluster", *vmId))
	}

	return vmResponse, &nodeName, nil
}

func (vmService *VmServiceImpl) GetVm(nodeName *string, vmId *string) (*proxmoxTypes.QemuResponse, *string, error) {
	if nodeName == nil {
		return vmService.SearchVmById(vmId)
	}
	response, responseError := vmService.FindVmByNodeWithId(nodeName, vmId)
	return response, nodeName, responseError
}

func (vmService *VmServiceImpl) UpdatePowerState(model *proxmoxTypes.VmModel) error {
	if model == nil {
		return errors.New("could not update provided model, no model provided")
	}
	if model.NodeName.IsNull() {
		return errors.New("could not get power state of the provided vm, node name was null")
	}
	if model.VmId.IsNull() {
		return errors.New("could not get power state of the provided vm, vm id was null")
	}

	status, getStatusError := vmService.proxmoxClient.GetVmStatus(model.NodeName.ValueStringPointer(), model.VmId.ValueStringPointer())

	if getStatusError != nil {
		return getStatusError
	}

	model.PowerState = types.StringValue(status)

	return nil
}

func (vmService *VmServiceImpl) CreateVm(plan *proxmoxTypes.VmModel) error {
	//TODO: detect need for cloudinit disk
	qemuVmCreationRequest := vmService.CreateVmRequest(plan, true, true)

	upid, vmCreationError := vmService.proxmoxClient.CreateVm(qemuVmCreationRequest, plan.NodeName.ValueString())

	if vmCreationError != nil {
		return errors.New(fmt.Sprintf("Failed to create proxmox vm, error response received: %s", vmCreationError.Error()))
	}

	taskCompletionError := vmService.taskService.WaitForTaskCompletion(plan.NodeName.ValueStringPointer(), upid)

	if taskCompletionError != nil {
		return errors.New(fmt.Sprintf("Creation of requested VM failed: %s", taskCompletionError.Error()))
	}
	return nil
}

func (vmService *VmServiceImpl) MatchVmPowerState(plan *proxmoxTypes.VmModel, currentState *proxmoxTypes.VmModel) error {
	if plan.PowerState.ValueString() == "running" && currentState.PowerState.ValueString() != "running" {
		startVmError := vmService.StartVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())
		if startVmError != nil {
			return startVmError
		}
		updatePowerStateError := vmService.UpdatePowerState(currentState)
		if updatePowerStateError != nil {
			return updatePowerStateError
		}
	} else if plan.PowerState.ValueString() == "stopped" && currentState.PowerState.ValueString() != "stopped" {
		stopVmError := vmService.ShutdownVm(plan.NodeName.ValueStringPointer(), plan.VmId.ValueStringPointer())
		if stopVmError != nil {
			return stopVmError
		}
		updatePowerStateError := vmService.UpdatePowerState(currentState)
		if updatePowerStateError != nil {
			return updatePowerStateError
		}
	}
	return nil
}

func (vmService *VmServiceImpl) DeleteVm(nodeName *string, vmId *string) error {

	vmStatus, getStatusError := vmService.proxmoxClient.GetVmStatus(nodeName, vmId)

	if getStatusError != nil {
		return getStatusError
	}

	if vmStatus != "stopped" {
		shutdownError := vmService.ShutdownVm(nodeName, vmId)
		if shutdownError != nil {
			return shutdownError
		}
	}

	upid, vmDeletionError := vmService.proxmoxClient.DeleteVmById(nodeName, vmId)

	if vmDeletionError != nil {
		return vmDeletionError
	}

	taskCompletionError := vmService.taskService.WaitForTaskCompletion(nodeName, upid)

	if taskCompletionError != nil {
		return taskCompletionError
	}
	return nil
}

func (vmService *VmServiceImpl) UpdateVm(plan *proxmoxTypes.VmModel, nodeName *string, vmId *string) error {
	qemuVmCreationRequest := vmService.CreateVmRequest(plan, false, false)

	upid, updateVmError := vmService.proxmoxClient.UpdateVm(qemuVmCreationRequest, nodeName, vmId)

	if updateVmError != nil {
		return updateVmError
	}

	waitForTaskCompletionError := vmService.taskService.WaitForTaskCompletion(nodeName, upid)

	if waitForTaskCompletionError != nil {
		return waitForTaskCompletionError
	}

	return nil
}

func (vmService *VmServiceImpl) MigrateVm(currentNode *string, newNode *string, vmId *string) error {
	upid, migrateVmError := vmService.proxmoxClient.MigrateVm(currentNode, newNode, vmId)
	if migrateVmError != nil {
		return migrateVmError
	}

	waitForTaskError := vmService.taskService.WaitForTaskCompletion(currentNode, upid)

	if waitForTaskError != nil {
		return waitForTaskError
	}
	return nil
}
