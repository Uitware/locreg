package parser

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/viper"
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
		Name string `mapstructure:"name" default:"locreg-built-image"`
		Tag  string `mapstructure:"tag"` // Set a git SHA if not peresent default to latest
	} `mapstructure:"image"`
	Tunnel struct {
		Provider struct {
			Ngrok struct {
				Name        string `mapstructure:"name" default:"locreg-ngrok"`
				Image       string `mapstructure:"image" default:"ngrok/ngrok"`
				Tag         string `mapstructure:"tag" default:"latest"`
				Port        int    `mapstructure:"port" default:"4040"`
				NetworkName string `mapstructure:"networkName" default:"locreg-ngrok"`
			} `mapstructure:"ngrok"`
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
					IPAddress     struct {
						Type  string     `mapstructure:"type" default:"Public"`
						Ports []struct { // should be set to default dynamically because it is a slice of structs
							Port     int    `mapstructure:"port"`
							Protocol string `mapstructure:"protocol"`
						} `mapstructure:"ports"`
					} `mapstructure:"ipAddress"`
					Resources struct {
						Requests struct {
							CPU    float64 `mapstructure:"cpu" default:"1.0"`
							Memory float64 `mapstructure:"memory" default:"1.5"`
						} `mapstructure:"requests"`
					} `mapstructure:"resources"`
				} `mapstructure:"containerInstance"`
			} `mapstructure:"azure"`
			AWS struct {
				Region string `mapstructure:"region" default:"us-east-1"`
				ECS    struct {
					ClusterName           string `mapstructure:"clusterName" default:"locreg-cluster"`
					ServiceName           string `mapstructure:"serviceName" default:"locreg-service"`
					ServiceContainerCount int    `mapstructure:"serviceContainerCount" default:"1"`
					TaskDefinition        struct {
						Family              string `mapstructure:"family" default:"locreg-task"`
						IAMRoleName         string `mapstructure:"awsRoleName" default:"locreg-role"`
						MemoryAllocation    int    `mapstructure:"memoryAllocation" default:"512"`
						CPUAllocation       int    `mapstructure:"cpuAllocation" default:"256"`
						ContainerDefinition struct {
							Name         string `mapstructure:"name" default:"locreg-container"`
							PortMappings []struct {
								ContainerPort int    `mapstructure:"containerPort"`
								HostPort      int    `mapstructure:"hostPort"`
								Protocol      string `mapstructure:"protocol"`
							} `mapstructure:"portMappings"`
						} `mapstructure:"containerDefinitions"`
					} `mapstructure:"taskDefinition"`
				} `mapstructure:"ecs"`
				VPC struct {
					CIDRBlock string `mapstructure:"cidrBlock" default:"10.10.0.0/16"`
					Subnet    struct {
						CIDRBlock string `mapstructure:"cidrBlock" default:"10.10.10.0/24"`
					} `mapstructure:"subnet"`
				} `mapstructure:"vpc"`
			} `mapstructure:"aws"`
		} `mapstructure:"provider"`
	} `mapstructure:"deploy"`
	Tags map[string]*string `mapstructure:"tags"`
}

func LoadConfig(filePath string) (*Config, error) {
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("❌ error reading config file: %w", err)
	}

	setStructDefaults(&Config{}, "") // Set default values based on `default` tag in struct fields
	setDynamicDefaults()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("❌ error unmarshaling config file: %w", err)
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
// It's a recursive function that sets default values for nested structs
// values are only set to providers whose name specified in the config file Config struct
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
			if isInConfig(key) {
				setStructDefaults(field.Addr().Interface(), key)
			}
		} else {
			if defaultValue, ok := structField.Tag.Lookup("default"); ok {
				switch field.Kind() {
				case reflect.String:
					viper.SetDefault(key, defaultValue)
				case reflect.Int:
					if v, err := strconv.Atoi(defaultValue); err == nil {
						viper.SetDefault(key, v)
					} else {
						panic(err)
					}
				case reflect.Float64:
					if v, err := strconv.ParseFloat(defaultValue, 64); err == nil {
						viper.SetDefault(key, v)
					} else {
						panic(err)
					}
				case reflect.Bool:
					if v, err := strconv.ParseBool(defaultValue); err == nil {
						viper.SetDefault(key, v)
					} else {
						panic(err)
					}
				default:
					panic("unhandled default case")
				}
			}
		}
	}
}

func setDynamicDefaults() {
	tags := viper.Get("tags")
	if tags == false {
		viper.Set("tags", map[string]*string{})
	}

	viper.SetDefault("registry.username", GenerateRandomString(36))
	viper.SetDefault("registry.password", GenerateRandomString(36))
	viper.SetDefault("deploy.provider.azure.appService.name", fmt.Sprintf("locregappservice%s", GenerateRandomString(8)))
	viper.SetDefault("deploy.provider.azure.containerInstance.ipAddress.ports", []map[string]interface{}{
		{
			"port":     80,
			"protocol": "TCP",
		},
	})
	viper.SetDefault("deploy.provider.aws.ecs.taskDefinition.containerDefinitions.portMappings", []map[string]interface{}{
		{
			"containerPort": 80,
			"hostPort":      80,
			"protocol":      "tcp",
		},
	})

	viper.SetDefault("image.tag", getGitSHA())
}

func getGitSHA() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "latest"
	}
	return strings.TrimSpace(string(output))
}

func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// GetRegistryImage returns the registry image with the tag
func (config *Config) GetRegistryImage() string {
	return fmt.Sprintf("%s:%s", config.Image.Name, config.Image.Tag)
}

func (config *Config) IsNgrokConfigured() bool {
	ngrokConfig := config.Tunnel.Provider.Ngrok
	v := reflect.ValueOf(ngrokConfig)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()) {
			fmt.Printf("Field %s is not set\n", typeOfS.Field(i).Name)
			return false
		}
	}
	return true
}

// IsAppServiceSet checks if the App Service configuration is set in the config.
// If it is set returns true if not returns false
func (config *Config) IsAppServiceSet() bool {
	emptyAppServicePlan := Config{}.Deploy.Provider.Azure.AppServicePlan
	emptyAppService := Config{}.Deploy.Provider.Azure.AppService
	return config.Deploy.Provider.Azure.AppServicePlan != emptyAppServicePlan &&
		config.Deploy.Provider.Azure.AppService != emptyAppService
}

// IsContainerInstanceSet checks if the Container Instance configuration is set in the config.
// If it is set returns true if not returns false
func (config *Config) IsContainerInstanceSet() bool {
	emptyContainerInstance := Config{}.Deploy.Provider.Azure.ContainerInstance
	// DeepEqual must be used because ContainerInstance contains a slice of structs inside it
	return !reflect.DeepEqual(config.Deploy.Provider.Azure.ContainerInstance, emptyContainerInstance)
}

func isInConfig(key string) bool {
	if len(strings.Split(key, ".")) >= 5 {
		return true
	}
	for _, k := range viper.AllKeys() {
		if strings.Contains(k, strings.ToLower(key)) {
			return true
		}
	}
	viper.SetDefault(key, nil)
	return false
}
