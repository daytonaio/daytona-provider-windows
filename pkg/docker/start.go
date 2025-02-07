// Copyright 2024 Daytona Platforms Inc.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
)

func (d *DockerClient) StartWorkspace(opts *CreateWorkspaceOptions, daytonaDownloadUrl string) error {
	containerName := d.GetWorkspaceContainerName(opts.Workspace)
	c, err := d.apiClient.ContainerInspect(context.TODO(), containerName)
	if err != nil {
		return fmt.Errorf("failed to inspect container when starting project: %w", err)
	}

	if !c.State.Running {
		err = d.apiClient.ContainerStart(context.TODO(), containerName, container.StartOptions{})
		if err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}

		d.OpenWebUI(d.targetOptions.RemoteHostname, opts.LogWriter)

		err = d.WaitForWindowsBoot(c.ID, d.targetOptions.RemoteHostname)
		if err != nil {
			return err
		}
	}

	return nil
}
