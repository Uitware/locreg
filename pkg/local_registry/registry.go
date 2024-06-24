package local_registry

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
)

// Registry
func RunRegistry(dockerClient *client.Client) error {
	// run distribution registry container
	ctx := context.Background()
	registryPort := "5000"                                                   // Later should be taken from config and this variable deleted
	registryVersion := "latest"                                              // Later should be taken from config
	registryName := "my_registry"                                            // Later should be taken from config
	imageVersion := fmt.Sprintf("distribution/registry:%s", registryVersion) // Later should be taken from config

	// Will be created because it has a special type of nat.port
	// Create specifically formated string for port mapping
	port, err := nat.NewPort("tcp", registryPort)
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
