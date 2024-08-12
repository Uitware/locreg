package ngrok

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Uitware/locreg/pkg/local_registry"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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
	if !validateNgrokAuthtokens() {
		return
	}
	ctx := context.Background()
	containerImage := fmt.Sprintf("%v:%v", config.Tunnel.Provider.Ngrok.Image, config.Tunnel.Provider.Ngrok.Tag)
	port, err := nat.NewPort("tcp", "4040")
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
				HostPort: strconv.Itoa(config.Tunnel.Provider.Ngrok.Port),
			},
		},
	}

	networkID := getNetworkID(dockerClient, config.Tunnel.Provider.Ngrok.NetworkName)
	if networkID == "" {
		netResp, err := dockerClient.NetworkCreate(
			context.Background(),
			config.Tunnel.Provider.Ngrok.NetworkName,
			network.CreateOptions{},
		)
		if err != nil {
			log.Fatalf("❌ failed to create network: %v", err)
		}
		networkID = netResp.ID
	}
	// Create container
	imagePuller, err := dockerClient.ImagePull(ctx, containerImage, image.PullOptions{})
	if err != nil {
		log.Fatalf("❌ failed to pull ngrok image: %v", err)
	}
	if err := local_registry.PrintLog(imagePuller); err != nil {
		log.Fatalf("❌ failed to pull image: %v", err)
	}

	resp, err := dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: containerImage,
			Cmd: []string{
				"http",
				// Forward traffic to registry on registry port
				fmt.Sprintf("%v:%v", config.Registry.Name, "5000"),
			},
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
				networkID: {},
			},
		},
		nil,
		config.Tunnel.Provider.Ngrok.Name,
	)
	if err != nil {
		log.Fatalf("❌ failed to create container: %v", err)
	}

	if err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		defer errorCleanup(resp.ID, err)
		log.Printf("❌ failed to start ngrok container: %v", err)
	}
	if err = writeToProfile(resp.ID, strconv.Itoa(config.Tunnel.Provider.Ngrok.Port)); err != nil {
		defer errorCleanup(resp.ID, err)
		log.Fatalf("❌ failed to write to profile: %v", err)
	}
}

// writeToProfile writes the container ID and credentials to the profile file in TOML format
func writeToProfile(dockerID string, port string) error {
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
	// Get tunnel URL from ngrok container API
	for i := 0; i < 5; i++ {
		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/tunnels", port))
		if err != nil {
			time.Sleep(time.Duration(i) * time.Second) // Wait for 5 seconds before retrying
			continue
		}
		break
	}
	err = json.NewDecoder(resp.Body).Decode(&tunnelsResponse)
	if err != nil {
		return fmt.Errorf("❌ failed to decode response body: %w", err)
	}
	// write to profile
	profile.Tunnel = &parser.Tunnel{
		ContainerID: dockerID,
		URL:         tunnelsResponse.Tunnels[0].PublicURL,
	}
	if err := parser.SaveProfile(profile, profilePath); err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}
	return nil
}

func getNetworkID(dockerClient *client.Client, networkName string) string {
	resp, err := dockerClient.NetworkList(
		context.Background(),
		network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", networkName)),
		})
	if err != nil {
		log.Fatalf("❌ failed to list networks: %v", err)
	}
	if len(resp) == 0 {
		return ""
	}
	return resp[0].ID
}

// validateNgrokAuthtokens validates the ngrok authtoken
// make request to ngrok API to validate the token and check if it is a tunnel credential
// if so return true else return false
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
