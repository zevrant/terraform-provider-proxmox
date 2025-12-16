package services

import (
	"errors"
	"fmt"
	"terraform-provider-proxmox/proxmox_client"
	"time"
)

type TaskService interface {
	WaitForTaskCompletion(nodeName *string, taskUpid *string) error
}

type TaskServiceImpl struct {
	proxmoxClient proxmox_client.ProxmoxClient
}

func NewTaskService(proxmoxClient proxmox_client.ProxmoxClient) TaskService {
	taskService := TaskServiceImpl{
		proxmoxClient: proxmoxClient,
	}
	return &taskService
}

func (taskService *TaskServiceImpl) WaitForTaskCompletion(nodeName *string, taskUpid *string) error {
	if nodeName == nil {
		return errors.New("cannot wait for task completion if a node name is not provided")
	}

	if taskUpid == nil {
		return errors.New("cannot wait for task to complete if no task is provided")
	}

	for {

		taskStatus, getTaskStatusError := taskService.proxmoxClient.GetTaskStatusByUpid(nodeName, taskUpid)

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
