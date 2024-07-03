package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
	"locreg/pkg/parser"
	"locreg/pkg/providers/azure"
	"log"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy all",
	Short: "Destroy all resources described in the config file",
	Long:  `Destroy all resources described in the config file`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "config.yaml"
		err := destroyAllResources(configFilePath)
		if err != nil {
			log.Fatalf("Error destroying resources: %v", err)
		}
		fmt.Println("All resources destroyed successfully")
	},
}

func destroyAllResources(configFilePath string) error {

	config, err := parser.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	azure.Destroy(config)

	err = local_registry.DestroyLocalRegistry(config)
	if err != nil {
		log.Printf("Error deleting local registry: %v", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
