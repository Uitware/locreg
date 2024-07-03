package local_registry

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"locreg/pkg/parser"
)

// StopAndRemoveContainer stops and removes a Docker container
func StopAndRemoveContainer(containerName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx := context.Background()

	stopOptions := container.StopOptions{
		Timeout: nil, // Використовуємо стандартний таймаут
	}

	if err := cli.ContainerStop(ctx, containerName, stopOptions); err != nil {
		log.Printf("Unable to stop container %s: %s", containerName, err)
		// Продовжуємо виконання, навіть якщо контейнер не вдалося зупинити
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		log.Printf("Unable to remove container: %s", err)
		return err
	}

	fmt.Printf("Container %s stopped and removed\n", containerName)
	return nil
}

func DestroyLocalRegistry(config *parser.Config) error {
	return StopAndRemoveContainer(config.Registry.Name)
}
