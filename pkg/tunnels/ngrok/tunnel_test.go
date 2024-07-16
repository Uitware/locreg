package ngrok

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"locreg/pkg/local_registry"
	"locreg/pkg/parser"
	"locreg/test/locreg_testutils"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Cant get current directory")
	}
	dir = filepath.Join(dir, "..", "..", "..")
	return dir
}

// openDummyPort runs dummy registry
func openDummyPort(t *testing.T) {

	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%s", "127.0.0.1", strconv.Itoa(getConfig(t).Registry.Port))) // Listen on any available port
	if err != nil {
		t.Fatalf("Failed to open dummy port: %v", err)
	}

	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(func() {
		listener.Close()
	})
}

// ConfigAndProfile returns config
func getConfig(t *testing.T) *parser.Config {
	config, err := parser.LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "tunnel", "locreg.yaml"))
	if err != nil {
		t.Fatalf("❌ failed to load config: %v", err)
	}
	return config
}

func getProfile(t *testing.T) *parser.Profile {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		t.Fatalf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		t.Fatalf("❌ failed to load or create profile: %v", err)
	}
	return profile
}

func createTunnel(t *testing.T) {
	config := getConfig(t)
	RunNgrokTunnelContainer(config)
	openDummyPort(t)
	t.Cleanup(
		func() {
			dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				t.Fatalf("❌ failed to create Docker client: %v", err)
			}
			runningTestContainer, err := dockerClient.ContainerList(
				context.Background(),
				container.ListOptions{
					Filters: filters.NewArgs(
						filters.Arg("name", "locreg-ngrok"),
					),
				})
			if err != nil {
				t.Fatalf("❌ failed to list containers: %v", err)
			}
			// Check if the slice is not empty before trying to access its first element
			if len(runningTestContainer) > 0 {
				err = local_registry.StopAndRemoveContainer(runningTestContainer[0].ID)
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

func TestRunNgrokTunnelContainer(t *testing.T) {
	createTunnel(t)
	profile := getProfile(t)
	if locreg_testutils.DoesContainerExist(t, profile.Tunnel.ContainerID) {
		t.Log("✅ container exist")
		if !locreg_testutils.IsContainerRunning(t, profile.Tunnel.ContainerID) {
			t.Fatalf("❌ container is not running")
		} else {
			t.Log("✅ container is running")
		}
	} else {
		t.Fatal("❌ container does not exist")
	}
}

func TestRunNgrokTunnelEnvVariables(t *testing.T) {
	t.Setenv("NGROK_AUTHTOKEN", "")
	if validateNgrokAuthtokens() {
		t.Fatalf("❌ NGROK_AUTHTOKEN is not being validated for beeing empty")
	} else {
		t.Log("✅ NGROK_AUTHTOKEN is validated")
	}

	t.Setenv("NGROK_AUTHTOKEN", "testasdawdwd4@Y#*GEehbfrnuqhd23yrg")
	if validateNgrokAuthtokens() {
		t.Fatalf("❌ NGROK_AUTHTOKEN is not validated")
	} else {
		t.Log("✅ NGROK_AUTHTOKEN is validated")
	}
}
