package ngrok

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"locreg/pkg/parser"
	"log"
	"net/http"
	"os"
	"time"
)

type Tunnels struct {
	Tunnels []struct {
		Name      string `json:"ID"`
		PublicURL string `json:"public_url"`
	} `json:"tunnels"`
}

// RunNgrokTunnelContainer runs a Docker container with ngrok image for tunneling local registry
func RunNgrokTunnelContainer(config *parser.Config) {
	ctx := context.Background()
	containerImage := "ngrok/ngrok:latest"
	containerPort := "4040"
	port, err := nat.NewPort("tcp", containerPort)
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ failed to run on port: %v", err)
	}
	portBindings := nat.PortMap{ // Container port bindings
		port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: containerPort,
			},
		},
	}

	if err != nil {
		log.Fatalf("❌ failed to create Docker client: %v", err)
	}
	networkId := getNetworkId(dockerClient)

	if networkId == "" {
		netResp, err := dockerClient.NetworkCreate(context.Background(), "locreg-ngrok", network.CreateOptions{})
		if err != nil {
			log.Fatalf("❌ failed to create network: %v", err)
		}
		networkId = netResp.ID
	}
	// Create container
	imagePuller, err := dockerClient.ImagePull(ctx, containerImage, image.PullOptions{})
	if err != nil {
		log.Fatalf("❌ failed to pull ngrok image: %v", err)
	}
	defer imagePuller.Close()
	io.Copy(os.Stdout, imagePuller)

	resp, err := dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: containerImage,
			Cmd:   []string{"http", fmt.Sprintf("%v:%v", config.Registry.Name, "5000")},
			Env: []string{
				"NGROK_AUTHTOKEN=" + os.Getenv("NGROK_AUTHTOKEN"),
			},
			ExposedPorts: nat.PortSet{
				port: struct{}{},
			},
		},
		&container.HostConfig{
			PortBindings: portBindings, // expose port 4040 for accessing ngrok api
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkId: {},
			},
		},
		nil,
		"locreg-ngrok",
	)
	err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		defer errorCleanup(resp.ID, err)
		log.Printf("❌ failed to start ngrok container: %v", err)
	}
	if err = writeToProfile(resp.ID); err != nil {
		defer errorCleanup(resp.ID, err)
		log.Fatalf("❌ failed to write to profile: %v", err)
	}
}

// writeToProfile writes the container ID and credentials to the profile file in TOML format
func writeToProfile(dockerId string) error {
	var tunnelsResponse Tunnels
	var resp *http.Response
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}
	// Get tunnel URL from ngrok container  API
	for i := 0; i < 5; i++ {
		resp, err = http.Get("http://127.0.0.1:4040/api/tunnels")
		if err != nil {
			time.Sleep(time.Duration(i) * time.Second) // Wait for 5 seconds before retrying
			continue
		}
		break
	}
	err = json.NewDecoder(resp.Body).Decode(&tunnelsResponse)
	if err != nil {
		return fmt.Errorf("❌ failed to decode response body: %v", err)
	}
	// write to profile
	profile.Tunnel.ContainerID = dockerId
	profile.Tunnel.URL = tunnelsResponse.Tunnels[0].PublicURL
	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to write profile: %w", err)
	}

	return nil
}

func getNetworkId(dockerClient *client.Client) string {
	resp, err := dockerClient.NetworkList(
		context.Background(),
		network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", "locreg-ngrok")),
		})
	if err != nil {
		log.Fatalf("❌ failed to list networks: %v", err)
	}
	if len(resp) == 0 {
		return ""
	}
	return resp[0].ID
}

func validateNgrokAuthtokens() bool {

	return false
}
