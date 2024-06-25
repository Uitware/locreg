package local_registry

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

func InitCommand() error {
	// Setup local registry
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	return RunRegistry(cli)
}

func BuildCommand(dir string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	return imageBuildAndPush(cli, dir)
}
func imageBuildAndPush(dockerClient *client.Client, dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	// TODO use config to set tag and port from runRegistry
	ImageTagString := "localhost:5000/test:latest"
	authConfig := registry.AuthConfig{
		Username:      "test",
		Password:      "test",
		ServerAddress: "http://127.0.0.1:5000",
	}

	defer cancel()

	tar, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{ImageTagString},
		Remove:     true,
	}
	res, err := dockerClient.ImageBuild(ctx, tar, opts)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	// Login into registry and push image
	_, err = dockerClient.RegistryLogin(ctx, authConfig)
	if err != nil {
		return err
	}

	encodedJSON, _ := json.Marshal(authConfig)
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	pushResponse, err := dockerClient.ImagePush(ctx, ImageTagString, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		fmt.Printf("Failed to push image: %v\n", err)
		return nil
	}
	defer func(pushResponse io.ReadCloser) {
		err := pushResponse.Close()
		if err != nil {

		}
	}(pushResponse)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	return printLog(pushResponse)
}

func printLog(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	if err := json.Unmarshal([]byte(lastLine), errLine); err == nil && errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// TODO - move usage to a separate file (where the short description resides)
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  locreg build [directory]  - Build the Docker image from the specified directory (default is current directory)")
}
