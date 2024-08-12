package local_registry

import (
	"context"
	"fmt"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/Uitware/locreg/test/locreg_testutils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
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
			runningTestContainer, err := dockerClient.ContainerList(
				context.Background(),
				container.ListOptions{
					Filters: filters.NewArgs(
						filters.Arg("name", config.Registry.Name),
					),
				})
			if err != nil {
				t.Fatalf("❌ failed to list containers: %v", err)
			}
			// Check if the slice is not empty before trying to access its first element
			if len(runningTestContainer) > 0 {
				err = StopAndRemoveContainer(runningTestContainer[0].ID)
				if err != nil {
					t.Fatalf(
						"❌ failed to stop and remove container. If in CI you may ignore it else delete it manually: %v",
						err)
				}
			} else {
				t.Log("No running containers found")
			}
		})
}

func isLocalRegistryAccessible(t *testing.T) bool {
	config, err := parser.LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "registry", "locreg.yaml"))
	if err != nil {
		t.Errorf("❌ failed to load config: %v", err)
	}
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("❌ failed to create Docker client: %v", err)
	}
	for delay := 1; delay <= 5; delay++ {
		authResp, err := dockerClient.RegistryLogin(
			context.Background(),
			registry.AuthConfig{
				Username:      config.Registry.Username,
				Password:      config.Registry.Password,
				ServerAddress: fmt.Sprintf("localhost:%s", strconv.Itoa(config.Registry.Port)),
			})
		if err != nil {
			t.Logf("Debug Logs: %v", err)
		}
		if authResp.Status == "Login Succeeded" {
			return true
		}
		time.Sleep(time.Duration(delay*2) * time.Second) // Wait for the specified delay before the next attempt
	}
	return false
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
	if locreg_testutils.DoesContainerExist(t, profile.LocalRegistry.RegistryID) {
		t.Log("✅ container exist")
		if !locreg_testutils.IsContainerRunning(t, profile.LocalRegistry.RegistryID) {
			t.Fatalf("❌ container is not running")
		} else {
			t.Log("✅ container is running")
		}
	} else {
		t.Fatal("❌ container does not exist")
	}

	// Test case 3: Does container accessible locally
	if !isLocalRegistryAccessible(t) {
		t.Fatalf("❌ failed to access local registry")
	} else {
		t.Log("✅ local registry is accessible")
	}
}

func TestProfileFilling(t *testing.T) {
	setUpRegistry(t)
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		t.Fatalf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		t.Fatalf("❌ failed to load or create profile: %v", err)
	}

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

func TestContainerErrorCleanUp(t *testing.T) {
	setUpRegistry(t)

	profilePath, err := parser.GetProfilePath()
	if err != nil {
		t.Fatalf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		t.Fatalf("❌ failed to load or create profile: %v", err)
	}

	// Call errorCleanup
	testError := fmt.Errorf("this is a test error")
	errorCleanup(profile.LocalRegistry.RegistryID, &testError)
	// Check if the container still exists
	if !locreg_testutils.DoesContainerExist(t, profile.LocalRegistry.RegistryID) {
		t.Fatalf("❌ container still exists after cleanup")
	} else {
		t.Log("✅ container successfully cleaned up")
	}

	setUpRegistry(t)
	errorCleanup(profile.LocalRegistry.RegistryID, nil)
	// Check if the container still exists
	if locreg_testutils.DoesContainerExist(t, profile.LocalRegistry.RegistryID) {
		t.Log("✅ container successfully cleaned up")
	} else {
		t.Fatalf("❌ container still exists after cleanup")
	}
}
