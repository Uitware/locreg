package local_registry

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/crypto/bcrypt"
	"io"
	"locreg/pkg/parser"
	"log"
	"os"
)

// RunRegistry runs a local Docker registry container with configuration
func RunRegistry(dockerClient *client.Client, config *parser.Config) error {

	// Use configuration values
	ctx := context.Background()
	registryPort := fmt.Sprintf("%d", config.Registry.Port)
	imageVersion := fmt.Sprintf("docker.io/%s:%s", config.Registry.Image, config.Registry.Tag)
	containerPort := "5000"

	// Create specifically formatted string for port mapping
	port, err := nat.NewPort("tcp", containerPort)
	if err != nil {
		return fmt.Errorf("failed to create port: %w", err)
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
	log.Default()
	if err != nil {
		log.Default()
		return fmt.Errorf("failed to pull distribution image: %w", err)
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
		nil,
		nil,
		config.Registry.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to create distribution container: %w", err)
	}

	err = updateConfig(dockerClient, ctx, resp.ID, config.Registry.Username, config.Registry.Password)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	err = dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start distribution container: %w", err)
	}
	fmt.Printf("Container started with ID: %s\n", resp.ID)
	return nil
}

func updateConfig(
	dockerClient *client.Client,
	ctx context.Context,
	containerID, username, password string,
) error {
	// Password configuration for registry should be hashed using bcrypt
	//credsTarBuffer, err := prepareCreds()
	//if err != nil {
	//	return fmt.Errorf("fatal error: %w", err)
	//}
	// Config file path inside the container
	// Prepare credentials
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	creds := fmt.Sprintf("%s:%s\n", username, hashedPassword)
	credsTarBuffer, err := prepareCreds(creds, "htpasswd")
	// Docker config variables
	containerConfigPath := "/etc/docker/registry/config.yml"
	configUpdates := `
auth:  
  htpasswd:
    realm: basic-realm    
    path: /htpasswd
` // May want to use structs later

	//Read from config
	reader, _, err := dockerClient.CopyFromContainer(
		ctx,
		containerID,
		containerConfigPath,
	)
	if err != nil {
		return fmt.Errorf("copy from container: %w", err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)
	var yamlBytes []byte
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar file: %v", err)
		}
		if header.Name == "config.yml" {
			yamlBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("error reading config.yml: %v", err)
			}
			break
		}
	}
	// May use structs instead of this string manipulation
	fmt.Println(string(yamlBytes) + configUpdates)
	configTarBuffer, err := prepareCreds(string(yamlBytes)+configUpdates, "config.yml")

	// Write the updated config back to the container
	err = dockerClient.CopyToContainer(
		ctx,
		containerID,
		"/",
		credsTarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}
	err = dockerClient.CopyToContainer(
		ctx,
		containerID,
		"/etc/docker/registry/",
		configTarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

// prepareCreds creates a tar archive with the htpasswd file data inside it stored in same way as htpasswd -Bnb command does
func prepareCreds(fileContent, fileName string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	// Create a tar writer
	tw := tar.NewWriter(&buf)

	// Add a file to the tar archive
	hdr := &tar.Header{
		Name: fileName,
		Mode: 0600,
		Size: int64(len(fileContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}
	if _, err := tw.Write([]byte(fileContent)); err != nil {
		return nil, fmt.Errorf("failed to write file content to tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	return &buf, nil
}

//func prepareCreds(username, password string) (*bytes.Buffer, error) {
//	var buf bytes.Buffer
//
//	// Create a tar writer
//	tw := tar.NewWriter(&buf)
//
//	// Hash the password using bcrypt
//	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
//	if err != nil {
//		return nil, fmt.Errorf("failed to hash password: %w", err)
//	}
//
//	// Create the content similar to htpasswd -Bnb
//	fileContent := fmt.Sprintf("%s:%s\n", username, hashedPassword)
//
//	// Add a file to the tar archive
//	hdr := &tar.Header{
//		Name: "htpasswd",
//		Mode: 0600,
//		Size: int64(len(fileContent)),
//	}
//	if err := tw.WriteHeader(hdr); err != nil {
//		return nil, fmt.Errorf("failed to write tar header: %w", err)
//	}
//	if _, err := tw.Write([]byte(fileContent)); err != nil {
//		return nil, fmt.Errorf("failed to write file content to tar: %w", err)
//	}
//	if err := tw.Close(); err != nil {
//		return nil, fmt.Errorf("failed to close tar writer: %w", err)
//	}
//	return &buf, nil
//}

func InitCommand(configFilePath string) error {
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	return RunRegistry(cli, config)
}
