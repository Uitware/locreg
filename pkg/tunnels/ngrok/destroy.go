package ngrok

import (
	"fmt"
	"github.com/Uitware/locreg/pkg/local_registry"
	"github.com/Uitware/locreg/pkg/parser"
	"log"
)

func DestroyTunnel() error {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	if profile.Tunnel.ContainerID == "" {
		return fmt.Errorf("❌ no tunnel container running found found in profile")
	}
	err = local_registry.StopAndRemoveContainer(profile.Tunnel.ContainerID)
	if err != nil {
		return fmt.Errorf("❌ failed to stop or remove tunnel container: %w", err)

	}
	log.Printf("❌ Tunnel container with container ID %s terminated", profile.Tunnel.ContainerID)

	return nil
}

func errorCleanup(containerID string, cleanupErr error) {
	if cleanupErr == nil {
		return
	}
	err := local_registry.StopAndRemoveContainer(containerID)
	if err != nil {
		log.Fatalf("❌ failed to stop or remove container after error if in CI ignore else do this on your own: %v", err)
	}
}
