package cmd

import (
	"fmt"
	"locreg/pkg/local_registry"
	"locreg/pkg/parser"
	"locreg/pkg/providers/azure"
	"log"
	"os"

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

	profilePath, err := parser.GetProfilePath()
	if err != nil {
		return fmt.Errorf("❌ failed to get profile path: %w", err)
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		return fmt.Errorf("❌ failed to load or create profile: %w", err)
	}

	if profile.LocalRegistry != nil {
		local_registry.DestroyLocalRegistry()
		profile.LocalRegistry = nil
	}
	if profile.Tunnel != nil {
		ngrok.DestroyTunnel()
		profile.Tunnel = nil
	}
	if profile.CloudResource != nil {
		azure.Destroy()
		profile.CloudResource = nil
	}

	err = os.Remove(profilePath)
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
