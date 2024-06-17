package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/go-connections/nat"
	"io"
	"os"
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

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "build":
		dir := "."
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		if err := buildCommand(dir); err != nil {
			fmt.Println("Error:", err.Error())
		}
	case "init":
		if err := initCommand(); err != nil {
			fmt.Println("Error:", err.Error())
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()

	}
}

func initCommand() error {
	// Setup local registry
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	return runDistribution(cli)
}

func buildCommand(dir string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	return imageBuildAndPsuh(cli, dir)
}

func runDistribution(dockerClient *client.Client) error {
	ctx := context.Background()
	portBindings := nat.PortMap{
		"5000/tcp": []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: "5000",
			},
		},
	}

	reader, err := dockerClient.ImagePull(ctx, "distribution/registry", image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull distribution image: %w", err)
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {

		}
	}(reader)

	resp, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "distribution/registry",
		ExposedPorts: nat.PortSet{
			"5000/tcp": struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: portBindings,
	}, nil, nil, "my_registry")
	if err != nil {
		return fmt.Errorf("failed to create distribution container: %w", err)
	}
	if err := dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start distribution container: %w", err)
	}

	fmt.Printf("Container started with ID: %s\n", resp.ID)
	return nil
}

func imageBuildAndPsuh(dockerClient *client.Client, dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	ImageTagString := "localhost:5000/test:latest"
	authConfig := registry.AuthConfig{
		Username:      "test",
		Password:      "test",
		ServerAddress: "http://127.0.0.1:5000",
	}
	// unsuccessful test of using AuthConfig directly with ImagePush
	//authConfig := types.AuthConfig{
	//	Username:      "test",
	//	Password:      "test",
	//	ServerAddress: "http://127.0.0.1:5000",
	//}

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

	// Read the push response
	//_, err = io.Copy(os.Stdout, pushResponse)
	//if err != nil {
	//	fmt.Printf("Failed to read push response: %v\n", err)
	//	return nil
	//}

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

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  locreg build [directory]  - Build the Docker image from the specified directory (default is current directory)")
}
