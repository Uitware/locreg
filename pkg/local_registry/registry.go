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
	"log"
	"os"
)

// RunRegistry runs a local Docker registry container with configuration
func RunRegistry(dockerClient *client.Client, config *parser.Config) error {

	// Use configuration values
	ctx := context.Background()
	registryPort := fmt.Sprintf("%d", config.Registry.Port)
	imageVersion := fmt.Sprintf("docker.io/%s:%s", config.Registry.Image, config.Registry.Tag)
	containerPort := "5000"

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
	log.Default()
	if err != nil {
		log.Default()
		return fmt.Errorf("failed to pull distribution image: %w", err)
	}
	defer imagePuller.Close()
	io.Copy(os.Stdout, imagePuller)

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
		config.Registry.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to create distribution container: %w", err)
	}
	config.Registry.Username = ""
	config.Registry.Password = ""

	err = updateConfig(dockerClient, ctx, resp.ID, config.Registry.Username, config.Registry.Password)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start distribution container: %w", err)
	}
	fmt.Printf("Container started with ID: %s\n", resp.ID)

	// TODO fix this part
	//if config.Registry.Username == "" || config.Registry.Password == "" {
	//	logsReader, err := dockerClient.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStderr: true, ShowStdout: true})
	//	if err != nil {
	//		return fmt.Errorf("failed to get password form container logs: %w", err)
	//	}
	//	logs, _ := io.ReadAll(logsReader)
	//	fmt.Println(string(logs))
	//}
	return nil
}

func InitCommand(configFilePath string) error {
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
