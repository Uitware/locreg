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

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate credentials of the local Docker registry",
	Long:  `This command rotates the credentials of the local Docker registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "config.yaml"
		err := local_registry.RotateCommand(configFilePath)
		if err != nil {
			fmt.Println("Error rotating registry credentials:", err)
		} else {
			fmt.Println("Credentials rotated successfully.")
		}
	},
}

func init() {
	registryCmd.AddCommand(rotateCmd)
	rootCmd.AddCommand(registryCmd)
}
