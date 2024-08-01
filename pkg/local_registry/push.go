package local_registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"locreg/pkg/parser"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

func BuildCommand(configFilePath string, dir string) error {
	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load config: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("❌ failed to create Docker client: %w", err)
	}
	profile, _ := parser.LoadProfileData()

	return imageBuildAndPush(cli, dir, config, profile)
}

func imageBuildAndPush(dockerClient *client.Client, dir string, config *parser.Config, profile *parser.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	ImageTagString := fmt.Sprintf("localhost:%d/%s:%s", config.Registry.Port, config.Image.Name, config.Image.Tag)
	authConfig := registry.AuthConfig{
		Username:      profile.LocalRegistry.Username,
		Password:      profile.LocalRegistry.Password,
		ServerAddress: fmt.Sprintf("http://127.0.0.1:%d", config.Registry.Port),
	}

	tar, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("❌ failed to create tar archive: %w", err)
	}

	buildOpts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{ImageTagString},
		Remove:     true,
	}
	buildResponse, err := dockerClient.ImageBuild(ctx, tar, buildOpts)
	if err != nil {
		return fmt.Errorf("❌ failed to build image: %w", err)
	}
	defer buildResponse.Body.Close()

	if err := PrintLog(buildResponse.Body); err != nil {
		return fmt.Errorf("❌ error during image build: %w", err)
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("❌ failed to encode auth config: %w", err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	pushResponse, err := dockerClient.ImagePush(ctx, ImageTagString, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		return fmt.Errorf("❌ failed to push image: %w", err)
	}
	defer pushResponse.Close()

	if err := PrintLog(pushResponse); err != nil {
		return fmt.Errorf("❌ error during image push: %w", err)
	}

	return nil
}
