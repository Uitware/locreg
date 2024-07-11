package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

type Profile struct {
	LocalRegistry struct {
		RegistryID string `toml:"registry_id"`
		Username   string `toml:"username"`
		Password   string `toml:"password"`
	} `toml:"local_registry"`
	Tunnel struct {
		URL string `toml:"tunnel_url"`
		PID int    `toml:"pid"`
	} `toml:"tunnel"`
	CloudResources struct {
		ResourceGroupName  string `toml:"resource_group_name"`
		AppServicePlanName string `toml:"app_service_plan_name"`
		AppServiceName     string `toml:"app_service_name"`
	} `toml:"cloud_resources"`
}

func GetProfilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("❌ failed to get home directory: %w", err)
	}
	profilePath := filepath.Join(homeDir, ".locreg")
	return profilePath, nil
}

func LoadOrCreateProfile(profilePath string) (*Profile, error) {
	var profile Profile
	if _, err := os.Stat(profilePath); err == nil {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("❌ failed to read profile file: %w", err)
		}
		err = toml.Unmarshal(data, &profile)
		if err != nil {
			return nil, fmt.Errorf("❌ failed to unmarshal profile file: %w", err)
		}
	} else {
		profile = Profile{}
	}
	return &profile, nil
}

func SaveProfile(profile *Profile, profilePath string) error {
	file, err := os.OpenFile(profilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("❌ failed to open profile file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(profile); err != nil {
		return fmt.Errorf("❌ failed to write to profile file: %w", err)
	}

	return nil
}
