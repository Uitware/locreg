package local_registry

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/bcrypt"
)

func updateConfig(
	dockerClient *client.Client,
	ctx context.Context,
	containerID, username, password string,
) error {
	// Prepare credentials
	// Password configuration for registry should be hashed using bcrypt
	credsTarBuffer := prepareCreds(username, password)
	// Write password file to container
	err := dockerClient.CopyToContainer(
		ctx,
		containerID,
		"/",
		credsTarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("❌ failed to copy to container: %w", err)
	}
	// Docker config variables
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
		"/etc/docker/registry/config.yml",
	)
	if err != nil {
		return fmt.Errorf("❌ copy from container: %w", err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)
	var yamlBytes []byte
	// iterate through the tar archive to find the config file if EOF reached stops use to remove header
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("❌ error reading tar file: %w", err)
		}
		if header.Name == "config.yml" {
			yamlBytes, err = io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("❌ error reading config.yml: %w", err)
			}
			break
		}
	}
	// May use structs instead of this string manipulation
	configTarBuffer, err := prepareTar(string(yamlBytes)+configUpdates, "config.yml")
	if err != nil {
		return fmt.Errorf("❌ failed copy to tar file into container: %w", err)
	}
	// Write config back to	container
	err = dockerClient.CopyToContainer(
		ctx,
		containerID,
		"/etc/docker/registry/",
		configTarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("❌ copy to container: %w", err)
	}

	return nil
}

func RotateCreds(
	dockerClient *client.Client,
	ctx context.Context,
	username, password, containerName string,
) error {
	credsTarBuffer := prepareCreds(username, password)
	// lookup container id from it's name
	containers, err := dockerClient.ContainerList(ctx,
		container.ListOptions{
			Filters: filters.NewArgs(
				filters.Arg("name", containerName),
			),
		})
	if err != nil {
		panic(err)
	}
	fmt.Println(containers[0].ID)

	// Write password file to container
	err = dockerClient.CopyToContainer(
		ctx,
		containers[0].ID,
		"/",
		credsTarBuffer,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return fmt.Errorf("❌ failed to copy to container: %w", err)
	}

	if err := dockerClient.ContainerRestart(ctx, containers[0].ID, container.StopOptions{}); err != nil {
		return fmt.Errorf("❌ failed to restart container: %w", err)
	}

	return nil
}

func prepareCreds(username, password string) *bytes.Buffer {
	// Prepare credentials
	// Password configuration for registry should be hashed using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ failed to hash password: %v", err)
	}
	credsTarBuffer, err := prepareTar(
		fmt.Sprintf("%s:%s\n", username, hashedPassword),
		"htpasswd",
	)
	if err != nil {
		log.Fatalf("❌ failed to prepare tar file: %v", err)
	}
	return credsTarBuffer
}

// prepareTar creates a tar archive with the htpasswd file data inside it stored in same way as htpasswd -Bnb command does
func prepareTar(fileContent, fileName string) (*bytes.Buffer, error) {
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
		return nil, fmt.Errorf("❌ failed to write tar header: %w", err)
	}
	if _, err := tw.Write([]byte(fileContent)); err != nil {
		return nil, fmt.Errorf("❌ failed to write file content to tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("❌ failed to close tar writer: %w", err)
	}
	return &buf, nil
}
