package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

type LocalRegistry struct {
	RegistryID string `toml:"registry_id"`
	Username   string `toml:"username"`
	Password   string `toml:"password"`
}

type Tunnel struct {
	URL         string `toml:"tunnel_url"`
	ContainerID string `toml:"tunnel_container_id"`
}

type AppService struct {
	ResourceGroupName  string `toml:"resource_group_name"`
	AppServicePlanName string `toml:"app_service_plan_name"`
	AppServiceName     string `toml:"app_service_name"`
}

type ContainerInstance struct {
	ResourceGroupName     string `toml:"resource_group_name"`
	ContainerInstanceName string `toml:"container_instance_name"`
}

type CloudResource struct {
	AppService        *AppService        `toml:"app_service,omitempty"`
	ContainerInstance *ContainerInstance `toml:"container_instance,omitempty"`
}

type Profile struct {
	LocalRegistry *LocalRegistry `toml:"local_registry,omitempty"`
	Tunnel        *Tunnel        `toml:"tunnel,omitempty"`
	CloudResource *CloudResource `toml:"cloud_resource,omitempty"`
}

// GetProfilePath returns the path to the profile file in the user's home directory
func GetProfilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("❌ failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".locreg"), nil
}

// LoadOrCreateProfile loads an existing profile or creates a new one if it doesn't exist
func LoadOrCreateProfile(profilePath string) (*Profile, error) {
	var profile Profile
	if _, err := os.Stat(profilePath); err == nil {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("❌ failed to read profile file: %w", err)
		}
		if err := toml.Unmarshal(data, &profile); err != nil {
			return nil, fmt.Errorf("❌ failed to unmarshal profile file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("❌ failed to check profile file: %w", err)
	}
	return &profile, nil
}

// SaveProfile saves the profile to the specified path
func SaveProfile(profile *Profile, profilePath string) error {
	file, err := os.OpenFile(profilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("❌ failed to open profile file: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(profile); err != nil {
		return fmt.Errorf("❌ failed to write to profile file: %w", err)
	}

	return nil
}
