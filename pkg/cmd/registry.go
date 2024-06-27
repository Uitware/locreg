package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Run a local Docker registry",
	Long:  `This command runs a local Docker registry using Docker.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "config.yaml"
		err := local_registry.InitCommand(configFilePath)
		if err != nil {
			fmt.Println("Error running registry:", err)
		} else {
			fmt.Println("Local registry is running.")
		}
	},
}

func init() {
	rootCmd.AddCommand(registryCmd)
}
