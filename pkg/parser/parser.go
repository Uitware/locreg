package parser

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"reflect"
)

type Config struct {
	Registry struct {
		Port     int    `yaml:"port" validate:"required"`
		Tag      string `yaml:"tag" validate:"required"`
		Name     string `yaml:"name" validate:"required"`
		Image    string `yaml:"image" validate:"required"`
		Username string `yaml:"username" validate:"required"`
		Password string `yaml:"password" validate:"required"`
	} `yaml:"registry"`
	Image struct {
		Name string `yaml:"name" validate:"required"`
		Tag  string `yaml:"tag" validate:"required"`
	} `yaml:"image"`
	Tunnel struct {
		Provider struct {
			Ngrok struct{} `yaml:"ngrok" validate:"required"`
		} `yaml:"provider" validate:"required"`
	} `yaml:"tunnel" validate:"required"`
	Deploy struct {
		Provider struct {
			Azure struct {
				Location       string `yaml:"location" validate:"required"`
				ResourceGroup  string `yaml:"resourceGroup" validate:"required"`
				AppServicePlan struct {
					Name string `yaml:"name" validate:"required"`
					Sku  struct {
						Name     string `yaml:"name" validate:"required"`
						Capacity int    `yaml:"capacity" validate:"required"`
						Tier     string `yaml:"tier" validate:"required"`
					} `yaml:"sku"`
					PlanProperties struct {
						Reserved bool `yaml:"reserved" validate:"required"`
					} `yaml:"planProperties"`
				} `yaml:"appServicePlan"`
				AppService struct {
					Name       string `yaml:"name" validate:"required"`
					SiteConfig struct {
						AlwaysOn                bool   `yaml:"alwaysOn" validate:"required"`
						DockerRegistryServerUrl string `yaml:"dockerRegistryServerUrl" validate:"required"`
						DockerImage             string `yaml:"dockerImage" validate:"required"`
						Tag                     string `yaml:"tag" validate:"required"`
					} `yaml:"siteConfig"`
				} `yaml:"appService"`
			} `yaml:"azure"`
		} `yaml:"provider"`
	} `yaml:"deploy"`
}

func LoadConfig(filePath string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config file: %v", err)
	}

	err = validateConfig(config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func validateConfig(config Config) error {
	v := reflect.ValueOf(config)
	return validateStruct(v)
}

func validateStruct(v reflect.Value) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if field.Kind() == reflect.Struct {
			if err := validateStruct(field); err != nil {
				return err
			}
		} else if tag := fieldType.Tag.Get("validate"); tag == "required" {
			if isEmptyValue(field) {
				return fmt.Errorf("missing required field: %s", fieldType.Name)
			}
		}
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
