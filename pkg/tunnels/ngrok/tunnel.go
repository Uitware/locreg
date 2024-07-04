package ngrok

import (
	"context"
	"fmt"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"locreg/pkg/parser"
	"log"
	"net/url"
	"os"
	"sync"
	"syscall"
)

func StartTunnel(configFilePath string) error {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Detach the process from the terminal
		// create child process and then detach from parent process
		pid, _, _ := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
		if pid < 0 {
			log.Println(fmt.Errorf("failed to fork process: %v", pid))
			return
		} else if pid > 0 {
			// If we got a good PID, then we call exit the parent process.
			log.Println(pid)
			return
		}

		ctx := context.Background()
		registryConfig, err := parser.LoadConfig(configFilePath)
		if err != nil {
			log.Println(fmt.Errorf("failed to load config: %w", err))
			return
		}
		err = runTunnel(ctx, registryConfig)
		if err != nil {
			log.Println(err)
			return
		}
	}()
	wg.Wait()
	return nil
}

func runTunnel(ctx context.Context, registryConfig *parser.Config) error {
	// Run tunnel
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

	// Load or create profile
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("failed to get profile path: %w", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("failed to load or create profile: %w", err)
	}

	profile.Tunnel.URL = tunnel.URL()
	profile.Tunnel.PID = os.Getpid()
	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	select {} // Keep the program running

}
