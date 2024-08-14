package cmd

import (
	"fmt"
	"github.com/Uitware/locreg/pkg/local_registry"
	"github.com/Uitware/locreg/pkg/parser"
	"log"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [directory]",
	Short: "Build and push a container image to the local registry",
	Long:  `Build a container image from the specified directory and push it to the local registry.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]
		configFilePath := "locreg.yaml"
		profile, _ := parser.LoadProfileData()
		err := local_registry.BuildCommand(configFilePath, dir)
		if err != nil {
			fmt.Println("❌ Error building and pushing image:", err)
		} else {
			fmt.Println("✅ Image successfully built and pushed.")
		}
		log.Print("Registry URL where image is located: ", profile.GetTunnelURL())
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	rootCmd.Root().CompletionOptions.DisableDefaultCmd = true
}
