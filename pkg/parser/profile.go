package parser

import (
	"fmt"
	"log"
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

type AzureCloudResource struct {
	AppService        *AppService        `toml:"app_service,omitempty"`
	ContainerInstance *ContainerInstance `toml:"container_instance,omitempty"`
}

type AWSCloudResource struct {
	ECSClusterARN     string `toml:"ecs_cluster_arn,omitempty"`
	TaskDefARN        string `toml:"task_def_arn,omitempty"`
	InternetGatewayId string `toml:"internet_gateway_arn,omitempty"`
	VPCId             string `toml:"vpc_id,omitempty"`
	ServiceARN        string `toml:"service_arn,omitempty"`
	SubnetId          string `toml:"subnet_id,omitempty"`
	RouteTableId      string `toml:"route_table_id,omitempty"`
}

type Profile struct {
	LocalRegistry      *LocalRegistry      `toml:"local_registry,omitempty"`
	Tunnel             *Tunnel             `toml:"tunnel,omitempty"`
	AzureCloudResource *AzureCloudResource `toml:"cloud_resource,omitempty"`
	AWSCloudResource   *AWSCloudResource   `toml:"aws_cloud_resource,omitempty"`
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
	} else if os.IsNotExist(err) {
		profile = Profile{}
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

// Save saves the profile to the user's home directory
// Newer version of SaveProfile function that avoids need of passing profilePath and profile as arguments
func (profile *Profile) Save() {
	profilePath, err := GetProfilePath()
	if err != nil {
		log.Fatal("❌ failed to get profile path: %w", err)
	}
	err = SaveProfile(profile, profilePath)
	if err != nil {
		log.Fatal("❌ failed to save profile: %w", err)
	}
}

// LoadProfileData loads the profile data from the user's home directory
// or creates what it is not found.
//
// Newer version of LoadOrCreateProfile function
func LoadProfileData() (*Profile, string) {
	profilePath, err := GetProfilePath()
	if err != nil {
		log.Printf("❌ Error getting profile path: %v", err)
		return nil, ""
	}

	profile, err := LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Printf("❌ Error loading or creating profile: %v", err)
		return nil, ""
	}
	return profile, profilePath
}

func (profile *Profile) GetTunnelURL() string {
	if profile.Tunnel == nil {
		log.Fatalf("❌ Tunnel does not exist")
	}
	return profile.Tunnel.URL
}
