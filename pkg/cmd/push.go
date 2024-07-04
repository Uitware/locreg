package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
)

var pushCmd = &cobra.Command{
	Use:   "push [directory]",
	Short: "Build and push a Docker image to the local registry",
	Long:  `This command builds a Docker image from the specified directory and pushes it to the local registry.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		configFilePath := "locreg.yaml"
		err := local_registry.BuildCommand(configFilePath, dir)
		if err != nil {
			fmt.Println("Error building and pushing image:", err)
		} else {
			fmt.Println("Image successfully built and pushed.")
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
