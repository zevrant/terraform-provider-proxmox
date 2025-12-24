package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-proxmox/proxmox_client"
	proxmox_types "terraform-provider-proxmox/types"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type NodeStorageService interface {
	QemuImageResponseDataToImage(
		imageName *string,
		storageName *string,
		imageResponse *proxmox_types.QemuImageResponseData) *proxmox_types.QemuImage
	ListImages(storageName *string, imageName *string) ([]proxmox_types.QemuImage, error)
	GetLatestImageFromImages(images []proxmox_types.QemuImage) *proxmox_types.QemuImage
}

type NodeStorageServiceImpl struct {
	client proxmox_client.ProxmoxClient
	tfCtx  context.Context
}

func NewNodeStorageService(client proxmox_client.ProxmoxClient, tfCtx context.Context) NodeStorageService {
	return &NodeStorageServiceImpl{
		client: client,
		tfCtx:  tfCtx,
	}
}

func (storageService *NodeStorageServiceImpl) QemuImageResponseDataToImage(
	imageName *string,
	storageName *string,
	imageResponse *proxmox_types.QemuImageResponseData) *proxmox_types.QemuImage {
	version := strings.Replace(imageResponse.Volid, fmt.Sprintf("%s:import/", *storageName), "", 1)
	version = strings.Replace(version, ".qcow2", "", 1)
	version = strings.Replace(version, *imageName, "", 1)
	version = strings.TrimPrefix(version, "-")

	image := proxmox_types.QemuImage{
		Name:        types.StringValue(*imageName),
		Version:     types.StringValue(version),
		StorageName: types.StringValue(*storageName),
	}
	return &image
}

func (storageService *NodeStorageServiceImpl) getFirstNodeSupportingStorage(storageName *string) (*string, error) {
	nodes, getNodesError := storageService.client.ListNodes()

	if getNodesError != nil {
		return nil, getNodesError
	}

	for _, node := range nodes.Data {
		nodeName := node.Node
		nodeStorages, getNodeStorageError := storageService.client.ListStorageDestinations(&nodeName)
		if getNodeStorageError != nil && len(nodes.Data) > 1 {
			tflog.Error(storageService.tfCtx, getNodeStorageError.Error())
			continue
		} else if getNodeStorageError != nil {
			return nil, getNodeStorageError
		}

		for _, storage := range nodeStorages.Data {
			if storage.Storage == *storageName {
				return &nodeName, nil
			}
		}
	}

	return nil, errors.New(fmt.Sprintf("storage with name %s could not be found on any node, please see logs for additional details.", *storageName))
}

func (storageService *NodeStorageServiceImpl) ListImages(storageName *string, imageName *string) ([]proxmox_types.QemuImage, error) {

	nodeName, getNodeForStorageError := storageService.getFirstNodeSupportingStorage(storageName)

	if getNodeForStorageError != nil {
		return nil, getNodeForStorageError
	}

	content, listContentError := storageService.client.ListStorageContent(nodeName, storageName)

	if listContentError != nil {
		return nil, listContentError
	}

	if content == nil || len(content.Data) == 0 {
		return nil, errors.New("no images returned from server")
	}

	var imagesWithName = make([]proxmox_types.QemuImage, 0)

	for _, image := range content.Data {
		if image.Content != "import" {
			continue
		}
		nameFromContent := strings.Replace(image.Volid, fmt.Sprintf("%s:import/", *storageName), "", 1)
		nameFromContent = strings.Replace(nameFromContent, ".qcow2", "", 1)
		nameFromContent = nameFromContent[0 : len(nameFromContent)-(len(nameFromContent)-len(*imageName))]
		if nameFromContent == *imageName {
			imagesWithName = append(imagesWithName, *storageService.QemuImageResponseDataToImage(imageName, storageName, &image))
		}
	}

	return imagesWithName, nil
}

func (storageService *NodeStorageServiceImpl) GetLatestImageFromImages(images []proxmox_types.QemuImage) *proxmox_types.QemuImage {
	var latest *proxmox_types.QemuImage = nil
	for _, image := range images {
		if latest == nil {
			latest = &image
		}
		value := storageService.compareSemanticVersion(latest.Version.ValueStringPointer(), image.Version.ValueStringPointer())

		if value < 0 {
			latest = &image
		}
	}
	return latest
}

func (storageService *NodeStorageServiceImpl) compareSemanticVersion(version1 *string, version2 *string) int {
	version1Parts := strings.Split(*version1, ".")
	version2Parts := strings.Split(*version2, ".")
	major1, _ := strconv.Atoi(version1Parts[0])
	major2, _ := strconv.Atoi(version2Parts[0])
	minor1, _ := strconv.Atoi(version1Parts[1])
	minor2, _ := strconv.Atoi(version2Parts[1])
	patch1, _ := strconv.Atoi(version1Parts[2])
	patch2, _ := strconv.Atoi(version2Parts[2])

	if major1 < major2 {
		return -1
	}
	if major1 > major2 {
		return 1
	}
	if minor1 < minor2 {
		return -1
	}
	if minor1 > minor2 {
		return 1
	}
	if patch1 < patch2 {
		return -1
	}
	if patch1 > patch2 {
		return 1
	}
	return 0
}
