package cmd

import (
	"fmt"
	"locreg/pkg/parser"
	"net"
	"net/http"
	"os"
	"os/exec"
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

func getProfile(t *testing.T) *parser.Profile {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		t.Errorf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		t.Errorf("❌ failed to load or create profile: %v", err)
	}
	// Workaround for some time. Profile parser need patch
	if profile.Tunnel.URL != "" {
		waitForTunnel(profile.Tunnel.URL, t, 10*time.Second)
	} else {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-timeout:
				t.Fatalf("❌ timeout while waiting for profile.Tunnel.URL to be non-empty")
			default:
				profile, err = parser.LoadOrCreateProfile(profilePath)
				if err != nil {
					t.Fatalf("❌ failed to load or create profile: %v", err)
				}
				if profile.Tunnel.URL != "" {
					break
				}
				time.Sleep(100 * time.Millisecond) // Sleep for a while before trying again
			}
		}
	}
	return profile
}

func createTunnel(t *testing.T) {
	err := exec.Command("go", "run", "../../../main.go", "tunnel").Run()
	if err != nil {
		t.Fatalf("Failed to run tunnel command: %v", err)
	}
	openDummyPort(t)
	t.Cleanup(func() {
		process, err := os.FindProcess(getProfile(t).Tunnel.PID)
		if err != nil {
			t.Fatalf("Error finding process: %v\n of tunnel. You nay ignore this in CI", err)
			return
		}

		err = process.Kill()
		if err != nil {
			fmt.Printf("Error killing process: %v\n. You nay ignore this in CI", err)
			return
		}
	})
}

// openDummyPort runs dummy registry
func openDummyPort(t *testing.T) {
	configFilePath := "locreg.yaml"
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
		return
	}
	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%s", "127.0.0.1", strconv.Itoa(config.Registry.Port))) // Listen on any available port
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

func waitForTunnel(tunnelURL string, t *testing.T, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isTunnelAlive(tunnelURL, t) {
			return true
		}
		time.Sleep(1 * time.Second) // Sleep for a while before trying again
	}
	return false
}

func isTunnelAlive(tunnelURL string, t *testing.T) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(tunnelURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	t.Log("Tunnel response status code: ", resp.StatusCode)
	return resp.StatusCode == http.StatusOK
}

func TestTunnelCmd(t *testing.T) {
	// Get test config file location
	workingDir := filepath.Join(getProjectRoot(), "test", "test_configs", "tunnel")
	if err := os.Chdir(workingDir); err != nil {
		t.Errorf("Failed to change directory to %s", workingDir)
	}

	// Create tunnel for testing
	createTunnel(t)
	profile := getProfile(t)

	if profile.Tunnel.PID == 0 {
		t.Fatal("❌ no tunnel PID found in profile. It was not created properly")
	} else {
		t.Logf("✅ tunnel PID: %v", profile.Tunnel.PID)
	}

	if profile.Tunnel.URL == "" {
		t.Fatal("❌ no tunnel URL found in profile. It was not created properly ")
	} else {
		t.Logf("✅ tunnel URL: %s", profile.Tunnel.URL)
	}

	if isTunnelAlive(profile.Tunnel.URL, t) == false {
		t.Errorf("❌ tunnel is not alive.")
	} else {
		t.Log("✅ tunnel successfully running")
	}
}
