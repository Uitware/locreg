package parser

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
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
		Name string `mapstructure:"name" default:"locreg-builded-image"`
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
					IpAddress     struct {
						Type  string     `mapstructure:"type" default:"Public"`
						Ports []struct { // should be set to default dynamically because it is a slice of structs
							Port     int    `mapstructure:"port"`
							Protocol string `mapstructure:"protocol"`
						} `mapstructure:"ports"`
					} `mapstructure:"ipAddress"`
					Resources struct {
						Requests struct {
							Cpu    float64 `mapstructure:"cpu" default:"1.0"`
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
	log.Print(viper.InConfig("deploy.provider.azure"))
	log.Print(viper.AllKeys()) // Use this keys to determine which default wars should be set

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
	log.Print(config)
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
			if isInConfig(key) {
				setStructDefaults(field.Addr().Interface(), key)
			}
		} else {
			if defaultValue, ok := structField.Tag.Lookup("default"); ok {
				switch field.Kind() {
				case reflect.String:
					log.Print(key, " ", defaultValue)
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

	viper.SetDefault("registry.username", generateRandomString(36))
	viper.SetDefault("registry.password", generateRandomString(36))
	viper.SetDefault("deploy.provider.azure.appService.name", fmt.Sprintf("locregappservice%s", generateRandomString(8)))
	viper.SetDefault("deploy.provider.azure.containerInstance.ipAddress.ports", []map[string]interface{}{
		{
			"port":     80,
			"protocol": "TCP",
		},
	})
	viper.SetDefault("image.tag", getGitSHA())
}

func getGitSHA() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 128 {
				return "latest"
			}
		}
		return "latest"
	}
	return strings.TrimSpace(string(output))
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
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
	log.Print(key)
	if len(strings.Split(key, ".")) >= 5 {
		return true
	}
	for _, k := range viper.AllKeys() {
		if strings.Contains(k, strings.ToLower(key)) {
			log.Print(k, " ", key)
			return true
		}
	}
	log.Print("Key not found: ", key)
	viper.SetDefault(key, nil)
	return false
}
