package local_registry

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"locreg/pkg/parser"
)

// RunRegistry runs a local Docker registry container with configuration
func RunRegistry(dockerClient *client.Client, config *parser.Config) error {
	// Use configuration values
	ctx := context.Background()
	registryPort := fmt.Sprintf("%d", config.Registry.Port)
	containerPort := "5000"
	registryVersion := config.Registry.Tag
	registryName := config.Registry.Name
	imageVersion := fmt.Sprintf("%s:%s", config.Registry.Image, registryVersion)

	// Create specifically formatted string for port mapping
	port, err := nat.NewPort("tcp", containerPort)
	if err != nil {
		return fmt.Errorf("failed to create port: %w", err)
	}
	portBindings := nat.PortMap{ // Container port bindings
		port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: registryPort,
			},
		},
	}

	imagePuller, err := dockerClient.ImagePull(ctx, imageVersion, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull distribution image: %w", err)
	}
	defer func(imagePuller io.ReadCloser) {
		err := imagePuller.Close()
		if err != nil {
			fmt.Printf("Failed to close image pull: %v\n", err)
		}
	}(imagePuller)

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: imageVersion, // Local registry image
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	},
		&container.HostConfig{
			PortBindings: portBindings,
		},
		nil,
		nil,
		registryName,
	)
	if err != nil {
		return fmt.Errorf("failed to create distribution container: %w", err)
	}

	err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start distribution container: %w", err)
	}
	fmt.Printf("Container started with ID: %s\n", resp.ID)
	return nil
}

func InitCommand(configFilePath string) error {
	// Завантаження конфігураційного файлу
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	return RunRegistry(cli, config)
}
