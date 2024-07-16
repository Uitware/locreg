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
	"strings"
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
	validateNgrokAuthtokens()
	ctx := context.Background()
	containerImage := "ngrok/ngrok:latest"
	containerPort := "4040"
	port, err := nat.NewPort("tcp", containerPort)
	if err != nil {
		log.Fatalf("❌ failed to run on port: %v", err)
	}
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("❌ failed to create Docker client: %v", err)
	}

	portBindings := nat.PortMap{ // Container port bindings
		port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: containerPort,
			},
		},
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
	var result map[string]interface{}
	desiredString := fmt.Sprintf(
		"The authentication you specified is actually a tunnel credential. Your credential: '%s'.",
		os.Getenv("NGROK_AUTHTOKEN"),
	)
	if os.Getenv("NGROK_AUTHTOKEN") == "" {
		log.Printf("❌ NGROK_AUTHTOKEN is not set")
		return false
	}
	// Make http request to ngrok API to validate the token
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", "https://api.ngrok.com/agent_ingresses", nil)
	if err != nil {
		log.Printf("❌ failed to create request: %v", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("NGROK_AUTHTOKEN"))
	req.Header.Set("Ngrok-Version", "2")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ failed to send request: %v", err)
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ failed to read response body: %v", err)
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("❌ failed to parse JSON: %v", err)
	}
	// Check if the response contains the explicit mention of the token being a tunnel credential
	if !strings.Contains(result["msg"].(string), desiredString) {
		log.Printf("❌ Token is not valid pleas use another one")
		return false
	}
	return true
}
