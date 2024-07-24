package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic("Cant get current directory")
	}
	dir = filepath.Join(dir, "..", "..")
	return dir
}

func TestDefaultValuesForRegistry(t *testing.T) {
	config, err := LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "parser", "locreg_without_creds.yaml"))
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	if config.Registry.Username == "" {
		t.Errorf("Username is not assigned a default random value")
	}
	if config.Registry.Password == "" {
		t.Errorf("Password is not assigned")
	}
}

func TestRetrievedValuesForRegistry(t *testing.T) {
	config, err := LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "parser", "locreg_with_creds.yaml"))
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	if config.Registry.Username != "test_username" {
		t.Errorf("Username is not retrieved correctly")
	}
	if config.Registry.Password != "test_password" {
		t.Errorf("Password is not retrieved correctly")
	}
}

func TestDefaultForOnlyOneValue(t *testing.T) {
	config, err := LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "parser", "locreg_with_only_username.yaml"))
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	if config.Registry.Username != "test_username" {
		t.Errorf("Username is not retrieved correctly")
	}
	if config.Registry.Password == "" {
		t.Errorf("Password is assigned")
	}

	// For password also
	config, err = LoadConfig(filepath.Join(getProjectRoot(), "test", "test_configs", "parser", "locreg_with_only_password.yaml"))
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	fmt.Print(config.Registry.Username)
	if config.Registry.Username == "" {
		t.Errorf("Username is not retrieved correctly")
	}
	if config.Registry.Password != "test_password" {
		t.Errorf("Password is not retrieved")
	}
}
