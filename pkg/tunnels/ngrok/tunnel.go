package ngrok

import (
	"context"
	"fmt"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"locreg/pkg/parser"
	"log"
	"net/url"
)

func StartTunnel(configFilePath string) error {
	ctx := context.Background()
	registryConfig, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	err = createTunnel(ctx, registryConfig)
	if err != nil {
		return err
	}
	return nil
}

func createTunnel(ctx context.Context, registryConfig *parser.Config) error {
	log.Println("Creating ngrok tunnel...")
	registryUrl := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", registryConfig.Registry.Port),
		Path:   "/", // This is the API version for Docker registry
	}
	tunnel, err := ngrok.ListenAndForward(
		ctx,
		&registryUrl,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(), // use NGROK_AUTHTOKEN environment variable
	)
	if err != nil {
		return fmt.Errorf("failed to start ngrok tunnel: %v", err)
	}
	log.Printf("ngrok tunnel for registry started: %s -> %s\n\n", tunnel.URL(), tunnel.ForwardsTo())
	select {} // Keep the program running
}
