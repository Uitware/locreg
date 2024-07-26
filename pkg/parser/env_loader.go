package parser

import (
	"github.com/joho/godotenv"
)

// LoadEnvVarsFromFile reads environment variables from the specified file and returns them as a map
func LoadEnvVarsFromFile(envFile string) (map[string]string, error) {
	envVars := make(map[string]string)
	envMap, err := godotenv.Read(envFile)
	if err != nil {
		return nil, err
	}
	for key, value := range envMap {
		envVars[key] = value
	}
	return envVars, nil
}
