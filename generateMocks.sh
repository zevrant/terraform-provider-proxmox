#!/bin/bash
#External Mocks
#mockgen github.com/diskfs/go-diskfs/filesystem FileSystem > services/filesystem_mock_test.go
#sed -i 's/package mock_filesystem/package services/g' services/filesystem_mock_test.go
#
#mockgen os FileInfo > services/fileInfo_mock_test.go
#sed -i 's/package mock_os/package services/g' services/fileInfo_mock_test.go
#
#mockgen github.com/diskfs/go-diskfs/filesystem File > services/filesystem_file_mock_test.go
#sed -i 's/package mock_filesystem/package services/g' services/filesystem_file_mock_test.go

#Internal Mocks

mockgen -source=proxmox_client/client.go -destination=proxmox_client/client_mock.go
sed -i 's/package mock_proxmox_client/package proxmox_client/g' proxmox_client/client_mock.go
sed -i 's/proxmox_client\.//g' proxmox_client/client_mock.go
sed -i 's~proxmox_client "terraform-provider-proxmox/proxmox_client"~~g' proxmox_client/client_mock.go


mockgen -source=services/proxmox_utils.go -destination=services/proxmox_utils_mock.go
sed -i 's/package mock_proxmox_client/package proxmox_client/g' services/proxmox_utils_mock.go
sed -i 's/mock_services\.//g' services/proxmox_utils_mock.go
sed -i 's~mock_services "terraform-provider-proxmox/proxmox_client"~~g' services/proxmox_utils_mock.go
sed -i 's/mock_services/services/g' services/proxmox_utils_mock.go