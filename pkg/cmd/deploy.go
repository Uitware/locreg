package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"locreg/pkg/providers/aws"
	"locreg/pkg/providers/azure"
	"locreg/pkg/providers/gcp"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [provider]",
	Short: "Deploy to a specified cloud provider",
	Long:  `Deploy your application to a specified cloud provider.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		switch provider {
		case "aws":
			aws.Deploy()
		case "azure":
			azure.Deploy()
		case "gcp":
			gcp.Deploy()
		default:
			fmt.Println("Unknown provider:", provider)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
