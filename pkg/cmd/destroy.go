package cmd

import (
	"fmt"
	"log"
	"os"

	"locreg/pkg/local_registry"
	"locreg/pkg/providers/azure"
	"locreg/pkg/tunnels/ngrok"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy all",
	Short: "Destroy all resources, defined in the locreg config file",
	Long:  `Destroy all resources, defined in the locreg.yaml config file: local registry, tunnel backend, applicatation backend`,
	Run: func(cmd *cobra.Command, args []string) {
		err := destroyAllResources()
		if err != nil {
			log.Fatalf("❌ Error destroying resources: %v", err)
		}
		fmt.Println("✅ All resources destroyed successfully")
	},
}

func destroyAllResources() error {

	local_registry.DestroyLocalRegistry()
	ngrok.DestroyTunnel()
	azure.Destroy()

	profilePath := os.ExpandEnv("$HOME/.locreg")
	err := os.Remove(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("❌ There are no resources created yet.")
		} else {
			return fmt.Errorf("❌ failed to remove profile: %w", err)
		}
	} else {
		fmt.Println("✅ Profile removed successfully")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
