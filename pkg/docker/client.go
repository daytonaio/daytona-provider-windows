// Copyright 2024 Daytona Platforms Inc.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	provider_types "github.com/daytonaio/daytona-provider-windows/pkg/types"
	"github.com/daytonaio/daytona/pkg/common"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/ssh"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type CreateWorkspaceOptions struct {
	Workspace           *models.Workspace
	WorkspaceDir        string
	ContainerRegistries common.ContainerRegistries
	LogWriter           io.Writer
	Gpc                 *models.GitProviderConfig
	SshClient           *ssh.Client
	BuilderImage        string
}

type IDockerClient interface {
	CreateWorkspace(opts *CreateWorkspaceOptions) error
	CreateTarget(target *models.Target, targetDir string, logWriter io.Writer, sshClient *ssh.Client) error

	DestroyWorkspace(workspace *models.Workspace, workspaceDir string, sshClient *ssh.Client) error
	DestroyTarget(target *models.Target, targetDir string, sshClient *ssh.Client) error

	StartWorkspace(opts *CreateWorkspaceOptions, daytonaDownloadUrl string) error
	StopWorkspace(workspace *models.Workspace, logWriter io.Writer) error

	GetWorkspaceProviderMetadata(workspace *models.Workspace) (string, error)
	GetTargetProviderMetadata(t *models.Target) (string, error)

	GetWorkspaceContainerName(workspace *models.Workspace) string
	GetWorkspaceVolumeName(workspace *models.Workspace) string
	GetContainerLogs(containerName string, logWriter io.Writer) error
	PullImage(imageName string, cr *models.ContainerRegistry, logWriter io.Writer) error
}

type DockerClientConfig struct {
	ApiClient     client.APIClient
	TargetOptions provider_types.TargetConfigOptions
}

func NewDockerClient(config DockerClientConfig) IDockerClient {
	return &DockerClient{
		apiClient:     config.ApiClient,
		targetOptions: config.TargetOptions,
	}
}

type DockerClient struct {
	apiClient     client.APIClient
	targetOptions provider_types.TargetConfigOptions
}

func (d *DockerClient) GetWorkspaceContainerName(workspace *models.Workspace) string {
	containers, err := d.apiClient.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("daytona.target.id=%s", workspace.TargetId)), filters.Arg("label", fmt.Sprintf("daytona.workspace.id=%s", workspace.Id))),
		All:     true,
	})
	if err != nil || len(containers) == 0 {
		return workspace.TargetId + "-" + workspace.Id
	}

	return containers[0].ID
}

func (d *DockerClient) GetWorkspaceVolumeName(workspace *models.Workspace) string {
	return workspace.TargetId + "-" + workspace.Id
}
func (d *DockerClient) OpenWebUI(hostname *string, logWriter io.Writer) {
	url := "http://localhost:8006"
	if hostname != nil {
		url = fmt.Sprintf("http://%s:8006", *hostname)
	}
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	case "darwin":
		err = exec.Command("open", url).Run()
	case "linux":
		err = exec.Command("xdg-open", url).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		logWriter.Write([]byte(fmt.Sprintf("Windows is started visit %s\n", url)))
	}

}

func (d *DockerClient) IsLocalWindowsTarget(providerName, options, runnerId string) bool {
	if providerName != "windows-provider" {
		return false
	}

	return !strings.Contains(options, "Remote Hostname") && runnerId == common.LOCAL_RUNNER_ID
}
