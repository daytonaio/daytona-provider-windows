// Copyright 2024 Daytona Platforms Inc.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/daytonaio/daytona/cmd/daytona/config"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/ports"
	"github.com/daytonaio/daytona/pkg/ssh"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

func (d *DockerClient) CreateTarget(target *models.Target, targetDir string, logWriter io.Writer, sshClient *ssh.Client) error {
	return nil
}

func (d *DockerClient) CreateWorkspace(opts *CreateWorkspaceOptions) error {
	ctx := context.TODO()
	mounts := []mount.Mount{}

	var availablePort *uint16
	portBindings := make(map[nat.Port][]nat.PortBinding)
	portBindings["22/tcp"] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "10022",
		},
	}
	portBindings["2222/tcp"] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "2222",
		},
	}

	uiPort := 8006
	for {
		if !ports.IsPortAvailable(uint16(uiPort)) {
			uiPort++
			continue
		}
		portBindings["8006/tcp"] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", uiPort),
			},
		}
		break
	}

	if d.IsLocalWindowsTarget(opts.Workspace.Target.TargetConfig.ProviderInfo.Name, opts.Workspace.Target.TargetConfig.Options, opts.Workspace.Target.TargetConfig.ProviderInfo.RunnerId) {
		p, err := ports.GetAvailableEphemeralPort()
		if err != nil {
			log.Error(err)
		} else {
			availablePort = &p
			portBindings["2280/tcp"] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", *availablePort),
				},
			}
		}
	}

	c, err := d.apiClient.ContainerCreate(ctx, GetContainerCreateConfig(opts.Workspace, availablePort), &container.HostConfig{
		Privileged: true,
		Mounts:     mounts,
		ExtraHosts: []string{
			"host.docker.internal:host-gateway",
		},
		PortBindings: portBindings,
		Resources: container.Resources{
			Devices: []container.DeviceMapping{
				{
					PathOnHost:      "/dev/kvm",
					PathInContainer: "/dev/kvm",
				},
				{
					PathOnHost:      "/dev/net/tun",
					PathInContainer: "/dev/net/tun",
				},
			},
		},
		CapAdd: []string{
			"NET_ADMIN",
			"SYS_ADMIN",
		},
	}, nil, nil, d.GetWorkspaceContainerName(opts.Workspace))
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = d.apiClient.ContainerStart(ctx, c.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	for {
		c, err := d.apiClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			return fmt.Errorf("failed to inspect container when creating project: %w", err)
		}

		if c.State.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}

	opts.LogWriter.Write([]byte("Installing Windows.....\n"))

	d.OpenWebUI(d.targetOptions.RemoteHostname, opts.LogWriter)

	err = d.WaitForWindowsBoot(c.ID, d.targetOptions.RemoteHostname)
	if err != nil {
		return fmt.Errorf("failed to wait for Windows to boot: %w", err)
	}

	sshClient, err := d.GetSshClient(d.targetOptions.RemoteHostname)
	if err != nil {
		return fmt.Errorf("failed to get SSH client: %w", err)
	}

	for key, env := range opts.Workspace.EnvVars {
		err = d.ExecuteCommand(fmt.Sprintf("setx %s \"%s\"", key, env), nil, sshClient)
		if err != nil {
			opts.LogWriter.Write([]byte(fmt.Sprintf("failed to set env variable %s to %s: %s\n", key, env, err.Error())))
		}
	}

	extraEnv := []string{
		fmt.Sprintf("setx HOME \"%s\"", "C:\\Users\\daytona"),
		"setx /M PATH \"%PATH%;C:\\Program Files\\Git\\bin\"",
	}
	for _, cmd := range extraEnv {
		err = d.ExecuteCommand(cmd, nil, sshClient)
		if err != nil {
			opts.LogWriter.Write([]byte(fmt.Sprintf("failed to execute command: %s, Error: %s\n", cmd, err.Error())))
		}
	}

	return nil
}

func GetContainerCreateConfig(workspace *models.Workspace, toolboxApiHostPort *uint16) *container.Config {
	envVars := []string{
		fmt.Sprintf("ARGUMENTS=%s", "-device e1000,netdev=net0  -netdev user,id=net0,hostfwd=tcp::22-:22,hostfwd=tcp::2222-:2222,hostfwd=tcp::2280-:2280"),
	}
	for key, value := range workspace.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	labels := map[string]string{
		"daytona.target.id":                workspace.TargetId,
		"daytona.workspace.id":             workspace.Id,
		"daytona.workspace.repository.url": workspace.Repository.Url,
	}

	if toolboxApiHostPort != nil {
		labels["daytona.toolbox.api.hostPort"] = fmt.Sprintf("%d", *toolboxApiHostPort)
	}

	exposedPorts := nat.PortSet{}
	if toolboxApiHostPort != nil {
		exposedPorts["2280/tcp"] = struct{}{}
	}

	exposedPorts["22/tcp"] = struct{}{}
	exposedPorts["2222/tcp"] = struct{}{}

	return &container.Config{
		Hostname: workspace.Id,
		Image:    "rutik7066/daytona-windows-container:latest",
		Labels:   labels,
		User:     "root",
		Entrypoint: []string{
			"/usr/bin/tini",
			"-s",
			"/run/entry.sh",
		},
		Env:          envVars,
		AttachStdout: true,
		AttachStderr: true,
		ExposedPorts: exposedPorts,
		StopTimeout:  &[]int{120}[0],
	}
}
