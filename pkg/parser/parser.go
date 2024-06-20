package parser

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
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
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
