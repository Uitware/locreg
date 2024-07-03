package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
	"locreg/pkg/providers/azure"
	"log"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy all",
	Short: "Destroy all resources described in the config file",
	Long:  `Destroy all resources described in the config file`,
	Run: func(cmd *cobra.Command, args []string) {

		err := destroyAllResources()
		if err != nil {
			log.Fatalf("Error destroying resources: %v", err)
		}
		fmt.Println("All resources destroyed successfully")
	},
}

func destroyAllResources() error {

	local_registry.DestroyLocalRegistry()
	azure.Destroy()

	return nil
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
