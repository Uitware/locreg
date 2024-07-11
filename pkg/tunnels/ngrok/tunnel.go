package ngrok

import (
	"context"
	"fmt"
	"locreg/pkg/parser"
	"log"
	"net/url"
	"os"
	"sync"
	"syscall"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

func StartTunnel(configFilePath string) error {
	if os.Getenv("NGROK_AUTHTOKEN") == "" || len(os.Getenv("NGROK_AUTHTOKEN")) != 49 {
		return fmt.Errorf("❌ NGROK_AUTHTOKEN environment variable is not set, or set incorrectly. Please " +
			"validate your ngrok authtoken")
	}
	if profile, _ := getProfile(); profile.Tunnel.PID != 0 {
		return fmt.Errorf("❌ tunnel is already running")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Detach the process from the terminal
		// create child process and then detach from parent process
		pid, _, _ := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
		if pid < 0 {
			log.Println(fmt.Errorf("❌ failed to fork process: %v", pid))
			return
		} else if pid > 0 {
			// If we got a good PID, then we call exit the parent process.
			log.Println(pid)
			return
		}

		ctx := context.Background()
		registryConfig, err := parser.LoadConfig(configFilePath)
		if err != nil {
			log.Println(fmt.Errorf("❌ failed to load config: %w", err))
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

// runTunnel creates a ngrok tunnel to the Docker registry in forked process for indefinite time
func runTunnel(ctx context.Context, registryConfig *parser.Config) error {
	// check if ngrok authtoken is set and is it valid size
	log.Println("🌐 Creating ngrok tunnel...")
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
		return fmt.Errorf("❌ failed to start ngrok tunnel: %v", err)
	}

	profile, profilePath := getProfile()

	profile.Tunnel.URL = tunnel.URL()
	profile.Tunnel.PID = os.Getpid()
	err = parser.SaveProfile(profile, profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to save profile: %w", err)
	}

	select {} // Keep the program running

}

func getProfile() (*parser.Profile, string) {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Fatalf("❌ failed to get profile path: %v", err)
	}
	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Fatalf("❌ failed to load or create profile: %v", err)
	}
	return profile, profilePath
}
