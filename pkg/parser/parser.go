package parser

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Registry struct {
		Port     int    `yaml:"port"`
		Tag      string `yaml:"tag"`
		Image    string `yaml:"image"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"Registry"`
	Deploy struct {
		Provider struct {
			Azure struct {
				Location        string `yaml:"location"`
				ResourceGroup   string `yaml:"resourceGroup"`
				AppsServicePlan struct {
					Name string `yaml:"name"`
					SKU  struct {
						Sku      string `yaml:"sku"`
						Capacity int    `yaml:"capacity"`
						Tier     string `yaml:"tier"`
					} `yaml:"SKU"`
					PlanProperties struct {
						Reserved bool `yaml:"reserved"`
					} `yaml:"PlanProperties"`
				} `yaml:"appsServicePlan"`
				AppService struct {
					Name       string `yaml:"name"`
					SiteConfig struct {
						AlwaysOn         bool   `yaml:"AlwaysOn"`
						DockregServerURL string `yaml:"dockregServerURL"`
						DockregUsername  string `yaml:"dockregUsername"`
						DockregPassword  string `yaml:"dockregPassword"`
						DockerImage      string `yaml:"dockerImage"`
						Tag              string `yaml:"tag"`
					} `yaml:"SiteConfig"`
				} `yaml:"appService"`
			} `yaml:"azure"`
		} `yaml:"provider"`
	} `yaml:"Deploy"`
}

func LoadConfig(filePath string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(filePath)
	if err == nil {
		// File exists, so we unmarshal it
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Config file not found, loading from environment variables")
	}
	return &config, nil
}
