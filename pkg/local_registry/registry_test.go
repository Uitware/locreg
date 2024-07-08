package local_registry

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"locreg/pkg/parser"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Cant get current directory")
	}
	dir = filepath.Join(dir, "..", "..")
	return dir
}

func setUpRegistry(t *testing.T) {
	config, err := parser.LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "registry", "locreg.yaml"))
	if err != nil {
		t.Errorf("❌ failed to load config: %v", err)
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("❌ failed to create Docker client: %v", err)
	}
	err = runRegistry(dockerClient, context.Background(), config)
	if err != nil {
		t.Fatalf("❌ failed to run registry: %v", err)
	}

	t.Cleanup(
		func() {
			t.Cleanup(func() {
				err := exec.Command("go", "run", "../../main.go", "destroy").Run()
				if err != nil {
					t.Errorf(
						"Failed to run destroy command: %v. If runned in CI no action needed else delete resources manualy",
						err,
					)
				}
			})
		})
}

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

func doesContainerExist(t *testing.T, containerID string) bool {
	return getContainers(t, containerID) != nil
}

func isContainerRunning(t *testing.T, containerID string) bool {
	containers := getContainers(t, containerID)
	if len(containers) == 0 {
		return false
	}
	return containers[0].State == "running"
}

func TestRunContainer(t *testing.T) {
	setUpRegistry(t)
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		t.Fatalf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		t.Fatalf("❌ failed to load or create profile: %v", err)
	}

	// Test case 1: Does container exist and running or not
	if doesContainerExist(t, profile.LocalRegistry.RegistryID) {
		t.Log("✅ container exist")
		if !isContainerRunning(t, profile.LocalRegistry.RegistryID) {
			t.Fatalf("❌ container is not running")
		} else {
			t.Log("✅ container is running")
		}
	} else {
		t.Error("❌ container does not exist")
	}

	// Test case 2: Does container creds exist in profile
	if profile.LocalRegistry.RegistryID == "" {
		t.Error("❌ failed to get container ID")
	}
	if profile.LocalRegistry.Username == "" {
		t.Error("❌ failed to get username")
	}
	if profile.LocalRegistry.Password == "" {
		t.Error("❌ failed to get username")
	}

}
