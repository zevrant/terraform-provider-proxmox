package vm

//func TestDiskServiceImpl_UpdateDisksFromQemuResponse(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	//realProxmoxUtils := services.NewProxmoxUtilService()
//	mockProxmoxUtils := services.NewMockProxmoxUtilService(ctrl)
//	mockProxmoxClient := proxmox_client.NewMockProxmoxClient(ctrl)
//	diskService := DiskServiceImpl{
//		proxmoxUtilsService: mockProxmoxUtils,
//		proxmoxClient:       mockProxmoxClient,
//	}
//	otherFields := map[string]interface{}{
//		"scsi0": "vm-os-storage:vm-9005-disk-0,iothread=1,size=50G",
//		"test":  "",
//		"scsi1": "cloudinit:myCloudinitStorage-9005-disk-0",
//	}
//	fieldMap := map[string]string{
//		"iothread":  "1",
//		"size":      "2048",
//		"aio":       "idk",
//		"replicate": "1",
//		"ro":        "0",
//		"ssd":       "0",
//		"backup":    "",
//		"discard":   "off",
//	}
//	mockProxmoxUtils.EXPECT().MapKeyValuePairsToMap(gomock.AssignableToTypeOf([]string{})).Times(1).Return(fieldMap)
//
//	vmModel := proxmoxTypes.VmModel{}
//	diskService.UpdateDisksFromQemuResponse(otherFields, &vmModel, nil)
//
//	//disks := vmModel.Disks
//
//}
