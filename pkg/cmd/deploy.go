package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/parser"
	"locreg/pkg/providers/aws"
	"locreg/pkg/providers/azure"
	"locreg/pkg/providers/gcp"
	"log"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy [provider]",
	Short: "Create a cloud resource and deploy your application",
	Long:  `Create a cloud provider's serverless container runtime resource and deploy your application.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]

		configFilePath := "locreg.yaml"
		config, err := parser.LoadConfig(configFilePath)
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		envFile, _ := cmd.Flags().GetString("env")
		// Load env variables from the provided file if specified
		var envVars map[string]string
		if envFile != "" {
			envVars, err = parser.LoadEnvVarsFromFile(envFile)
			if err != nil {
				log.Fatalf("❌ Error loading env file: %v", err)
			}
		}

		switch provider {
		case "aws":
			aws.Deploy()
		case "azure":
			azure.Deploy(config, envVars)

		case "gcp":
			gcp.Deploy()
		default:
			fmt.Println("❌ Unknown provider:", provider)
		}
	},
}

func init() {
	deployCmd.Flags().StringP("env", "e", "", "Path to the env file")
	rootCmd.AddCommand(deployCmd)
}
