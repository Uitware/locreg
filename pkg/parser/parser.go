package parser

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/spf13/viper"
)

type Config struct {
	Registry struct {
		Port     int    `mapstructure:"port"`
		Tag      string `mapstructure:"tag"`
		Name     string `mapstructure:"name"`
		Image    string `mapstructure:"image"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"registry"`
	Image struct {
		Name string `mapstructure:"name"`
		Tag  string `mapstructure:"tag"`
	} `mapstructure:"image"`
	Tunnel struct {
		Provider struct {
			Ngrok struct{} `mapstructure:"ngrok"`
		} `mapstructure:"provider"`
	} `mapstructure:"tunnel"`
	Deploy struct {
		Provider struct {
			Azure struct {
				Location       string `mapstructure:"location"`
				ResourceGroup  string `mapstructure:"resourceGroup"`
				AppServicePlan struct {
					Name string `mapstructure:"name"`
					Sku  struct {
						Name     string `mapstructure:"name"`
						Capacity int    `mapstructure:"capacity"`
						Tier     string `mapstructure:"tier"`
					} `mapstructure:"sku"`
					PlanProperties struct {
						Reserved bool `mapstructure:"reserved"`
					} `mapstructure:"planProperties"`
				} `mapstructure:"appServicePlan"`
				AppService struct {
					Name       string `mapstructure:"name"`
					SiteConfig struct {
						AlwaysOn bool `mapstructure:"alwaysOn"`
					} `mapstructure:"siteConfig"`
				} `mapstructure:"appService"`
				ContainerInstance struct {
					Name          string `mapstructure:"name"`
					OsType        string `mapstructure:"osType"`
					RestartPolicy string `mapstructure:"restartPolicy"`
					IpAddress     struct {
						Type  string `mapstructure:"type"`
						Ports []struct {
							Port     int    `mapstructure:"port"`
							Protocol string `mapstructure:"protocol"`
						} `mapstructure:"ports"`
					} `mapstructure:"ipAddress"`
					Resources struct {
						Requests struct {
							Cpu    float64 `mapstructure:"cpu"`
							Memory float64 `mapstructure:"memory"`
						} `mapstructure:"requests"`
					} `mapstructure:"resources"`
				} `mapstructure:"containerInstance"`
			} `mapstructure:"azure"`
		} `mapstructure:"provider"`
	} `mapstructure:"deploy"`
	Tags map[string]*string `mapstructure:"tags" `
}

func LoadConfig(filePath string) (*Config, error) {
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("❌ error reading config file: %v", err)
	}

	tags := viper.Get("tags")
	if tags == false {
		viper.Set("tags", map[string]*string{})
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("❌ error unmarshaling config file: %v", err)
	}

	if config.Tags == nil || config.Tags["managed-by-locreg"] == to.Ptr("false") {
		// If it is, set it to a default value
		defaultValue := "locreg"
		config.Tags = map[string]*string{
			"managed-by": &defaultValue,
		}
	}

	return &config, nil
}
