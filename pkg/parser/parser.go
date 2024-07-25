package parser

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
	"reflect"
)

type Config struct {
	Registry struct {
		Port     int    `mapstructure:"port" default:"5000"`
		Tag      string `mapstructure:"tag" default:"2"`
		Name     string `mapstructure:"name" default:"locreg-registry"`
		Image    string `mapstructure:"image" default:"registry"`
		Username string `mapstructure:"username"` // Set separately as should be unique each time
		Password string `mapstructure:"password"` // Set separately as should be unique each time
	} `mapstructure:"registry"`
	Image struct {
		Name string `mapstructure:"name" default:"app-image"`
		Tag  string `mapstructure:"tag" default:"latest"`
	} `mapstructure:"image"`
	Tunnel struct {
		Provider struct {
			Ngrok struct{} `mapstructure:"ngrok"`
		} `mapstructure:"provider"`
	} `mapstructure:"tunnel"`
	Deploy struct {
		Provider struct {
			Azure struct {
				Location       string `mapstructure:"location" default:"eastus"`
				ResourceGroup  string `mapstructure:"resourceGroup" default:"LocregResourceGroup"`
				AppServicePlan struct {
					Name string `mapstructure:"name" default:"LocregAppServicePlan"`
					Sku  struct {
						Name     string `mapstructure:"name" default:"F1"`
						Capacity int    `mapstructure:"capacity" default:"1"`
					} `mapstructure:"sku"`
					PlanProperties struct {
						Reserved bool `mapstructure:"reserved" default:"true"`
					} `mapstructure:"planProperties"`
				} `mapstructure:"appServicePlan"`
				AppService struct {
					Name       string `mapstructure:"name"` // Generated with random suffix
					SiteConfig struct {
						AlwaysOn bool `mapstructure:"alwaysOn" default:"false"`
					} `mapstructure:"siteConfig"`
				} `mapstructure:"appService"`
				ContainerInstance struct {
					Name          string `mapstructure:"name" default:"locreg-container"`
					OsType        string `mapstructure:"osType" default:"Linux"`
					RestartPolicy string `mapstructure:"restartPolicy" default:"Always"`
					IpAddress     struct {
						Type  string `mapstructure:"type" default:"Public"`
						Ports []struct {
							Port     int    `mapstructure:"port" default:"80""`
							Protocol string `mapstructure:"protocol" default:"TCP"`
						} `mapstructure:"ports"`
					} `mapstructure:"ipAddress"`
					Resources struct {
						Requests struct {
							Cpu    float64 `mapstructure:"cpu" default:"0.5"`
							Memory float64 `mapstructure:"memory" default:"1.5"`
						} `mapstructure:"requests"`
					} `mapstructure:"resources"`
				} `mapstructure:"containerInstance"`
			} `mapstructure:"azure"`
		} `mapstructure:"provider"`
	} `mapstructure:"deploy"`
	Tags map[string]*string `mapstructure:"tags"`
}

func LoadConfig(filePath string) (*Config, error) {
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("❌ error reading config file: %v", err)
	}

	setDynamicDefaults()
	setStructDefaults(&Config{}, "") // Set default values based on `default` tag in struct fields

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("❌ error unmarshaling config file: %v", err)
	}

	if config.Tags == nil {
		// If it is, set it to a default value
		defaultValue := "locreg"
		config.Tags = map[string]*string{
			"managed-by": &defaultValue,
		}
	}

	return &config, nil
}

// setStructDefaults sets default values based on `default` tag in struct fields
// It is a recursive function that sets default values for nested structs
func setStructDefaults(config interface{}, parentKey string) {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)
		key := structField.Tag.Get("mapstructure")

		if parentKey != "" {
			key = parentKey + "." + key
		}

		if field.Kind() == reflect.Struct {
			setStructDefaults(field.Addr().Interface(), key)
		}

		if defaultValue, ok := structField.Tag.Lookup("default"); ok {
			viper.SetDefault(key, defaultValue)
		}
	}
}

func setDynamicDefaults() {
	tags := viper.Get("tags")
	if tags == false {
		viper.Set("tags", map[string]*string{})
	}

	viper.SetDefault("registry.username", generateRandomString(36))
	viper.SetDefault("registry.password", generateRandomString(36))
	viper.SetDefault("deploy.provider.azure.appService.name", fmt.Sprintf("locregappservice%s", generateRandomString(8)))
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
