package ngrok

import (
	"fmt"
	"locreg/pkg/parser"
	"log"
	"syscall"
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

	if profile.Tunnel.PID == 0 {
		return fmt.Errorf("❌ no tunnel PID found in profile")
	}

	// Attempt to kill the process with the stored PID
	if err := syscall.Kill(profile.Tunnel.PID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("❌ failed to kill tunnel process: %w", err)
	}
	log.Printf("❌ Tunnel process with PID %d terminated", profile.Tunnel.PID)
	return nil
}
