package local_registry

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/bcrypt"
	"io"
)

func updateConfig(
	dockerClient *client.Client,
	ctx context.Context,
	containerID, username, password string,
) error {
	// Prepare credentials
	// Password configuration for registry should be hashed using bcrypt
	if username != "" || password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		creds := fmt.Sprintf("%s:%s\n", username, hashedPassword)
		credsTarBuffer, err := prepareTar(creds, "htpasswd")
		// Write password file to container
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
		return fmt.Errorf("copy from container: %w", err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)
	var yamlBytes []byte
	// iterate through the tar archive to find the config file if EOF reached stops use to remove header
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
	configTarBuffer, err := prepareTar(string(yamlBytes)+configUpdates, "config.yml")

	// Write config back to	container
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
