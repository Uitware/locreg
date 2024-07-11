package cmd

import (
	"fmt"
	"locreg/pkg/parser"
	"log"

	"locreg/pkg/providers/aws"
	"locreg/pkg/providers/azure"
	"locreg/pkg/providers/gcp"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [provider]",
	Short: "Сreate a cloud resource and deploy your application",
	Long:  `Сreate a cloud provider's serverless container runtime resource and deploy your application.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		switch provider {
		case "aws":
			aws.Deploy()
		case "azure":
			{
				configFilePath := "locreg.yaml"
				config, err := parser.LoadConfig(configFilePath)
				if err != nil {
					log.Fatalf("❌ Error loading config: %v", err)
				}
				azure.Deploy(config)
			}
		case "gcp":
			gcp.Deploy()
		default:
			fmt.Println("❌ Unknown provider:", provider)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
