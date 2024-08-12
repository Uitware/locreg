package locreg_testutils

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"testing"
)

func getContainers(t *testing.T, containerID string) []types.Container {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("❌ failed to create Docker client: %v", err)
	}
	containers, err := dockerClient.ContainerList(
		context.Background(),
		container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("id", containerID),
			),
		})
	if err != nil {
		t.Fatalf("❌ failed to list containers: %v", err)
	}
	return containers
}

func DoesContainerExist(t *testing.T, containerID string) bool {
	return getContainers(t, containerID) != nil
}

func IsContainerRunning(t *testing.T, containerID string) bool {
	containers := getContainers(t, containerID)
	if len(containers) == 0 {
		return false
	}
	return containers[0].State == "running"
}
