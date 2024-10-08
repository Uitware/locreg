package local_registry

import (
	"context"
	"fmt"
	"log"

	"github.com/Uitware/locreg/pkg/parser"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// StopAndRemoveContainer stops and removes a Docker container
func StopAndRemoveContainer(containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("❌ failed to create Docker client: %w", err)
	}

	ctx := context.Background()

	stopOptions := container.StopOptions{
		Timeout: nil, // Use default timeout
	}

	if err := cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		log.Printf("❌ Unable to stop container %s: %s", containerID, err)
		// Continue execution even if container couldn't be stopped
	}

	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		log.Printf("❌ Unable to remove container: %s", err)
		return err
	}

	fmt.Printf("✅ Container %s stopped and removed\n", containerID)
	return nil
}

// DestroyLocalRegistry stops and removes the local Docker registry based on the ID stored in the profile
func DestroyLocalRegistry() error {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	return StopAndRemoveContainer(profile.LocalRegistry.RegistryID)
}
