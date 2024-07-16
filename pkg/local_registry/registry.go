package local_registry

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"io"
	"locreg/pkg/parser"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// RunRegistry runs a local Docker registry container with configuration
func runRegistry(dockerClient *client.Client, ctx context.Context, config *parser.Config) error {
	// Use configuration values
	registryPort := fmt.Sprintf("%d", config.Registry.Port)
	imageVersion := fmt.Sprintf("docker.io/%s:%s", config.Registry.Image, config.Registry.Tag)
	containerPort := "5000"

	// Create specifically formatted string for port mapping
	port, err := nat.NewPort("tcp", containerPort)
	if err != nil {
		return fmt.Errorf("❌ failed to run on port: %w", err)
	}
	portBindings := nat.PortMap{ // Container port bindings
		port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: registryPort,
			},
		},
	}

	imagePuller, err := dockerClient.ImagePull(ctx, imageVersion, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("❌ failed to pull distribution image: %w", err)
	}
	defer imagePuller.Close()
	io.Copy(os.Stdout, imagePuller)

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: imageVersion, // Local registry image
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	},
		&container.HostConfig{
			PortBindings: portBindings,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				getNetworkId(dockerClient): {}, // connect container to ngrok network os it can be tunneled
			},
		},
		nil,
		config.Registry.Name,
	)
	if err != nil {
		return fmt.Errorf("❌ failed to create distribution container: %w", err)
	}

	err = updateConfig(dockerClient, ctx, resp.ID, config.Registry.Username, config.Registry.Password)
	if err != nil {
		defer errorCleanup(resp.ID, &err) // define postpone function to remove container if error occurs
		return fmt.Errorf("❌ failed to update config: %w", err)
	}

	err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		defer errorCleanup(resp.ID, &err) // define postpone function to remove container if error occurs
		return fmt.Errorf("❌ failed to start distribution container: %w", err)
	}
	fmt.Printf("✅ Container started with ID: %s\n", resp.ID)

	err = writeProfile(resp.ID, config.Registry.Username, config.Registry.Password)
	if err != nil {
		defer errorCleanup(resp.ID, &err) // define postpone function to remove container if error occurs
		return fmt.Errorf("❌ failed to write profile: %w", err)
	}
	return nil
}

func errorCleanup(containerID string, err *error) {
	if err == nil {
		return
	}
	if errDestroy := DestroyLocalRegistry(); errDestroy != nil {
		cleanupErr := StopAndRemoveContainer(containerID)
		if cleanupErr != nil {
			log.Fatalf("❌ Failed to remove container: %v. You will need to do this manualy", cleanupErr)
		}
	}
}

func InitCommand(configFilePath string) error {
	ctx := context.Background()
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load config: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("❌ failed to create Docker client: %w", err)
	}
	return runRegistry(cli, ctx, config)
}

func RotateCommand(configFilePath string) error {
	ctx := context.Background()
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load config: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("❌ failed to create Docker client: %w", err)
	}
	return RotateCreds(cli, ctx, config.Registry.Username, config.Registry.Password, config.Registry.Name)
}

func getNetworkId(dockerClient *client.Client) string {
	// retrieve network ID that was created for ngrok tunnel if it exists
	resp, err := dockerClient.NetworkList(
		context.Background(),
		network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", "locreg-ngrok")),
		})
	if err != nil {
		log.Fatalf("❌ failed to list networks: %v", err)
	}
	if len(resp) == 0 {
		netResp, err := dockerClient.NetworkCreate(context.Background(), "locreg-ngrok", network.CreateOptions{})
		if err != nil {
			log.Fatalf("❌ failed to create network: %v", err)
		}
		return netResp.ID
	}
	return resp[0].ID
}

// writeProfile writes the container ID and credentials to the profile file in TOML format
func writeProfile(containerID, username, password string) error {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	profile.LocalRegistry.RegistryID = containerID
	profile.LocalRegistry.Username = username
	profile.LocalRegistry.Password = password

	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}

	return nil
}
