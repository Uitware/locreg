package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
	"locreg/pkg/parser"
	"locreg/pkg/providers/azure"
	"locreg/pkg/tunnels/ngrok"
	"log"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy [resource|all]",
	Short: "Destroy specified resources or all resources defined in the locreg config file",
	Long:  `Destroy specified resources defined in the locreg.yaml config file, such as local registry, tunnel backend, cloud resources, or all resources.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resource := args[0]

		profile, profilePath := loadProfile()
		if profile == nil {
			log.Fatalf("❌ Failed to load profile.")
		}

		switch resource {
		case "registry":
			if profile.LocalRegistry != nil {
				local_registry.DestroyLocalRegistry()
				profile.LocalRegistry = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Registry destroyed successfully")
			}

		case "tunnel":
			if profile.Tunnel != nil {
				ngrok.DestroyTunnel()
				profile.Tunnel = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Tunnel destroyed successfully")
			}

		case "cloud":
			if profile.CloudResource != nil {
				azure.Destroy()
				profile.CloudResource = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Cloud resources destroyed successfully")
			}

		case "all":
			destroyAllResources(profile, profilePath)
			fmt.Println("✅ All resources destroyed successfully")

		default:
			fmt.Println("❌ Unknown resource:", resource)
		}
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}

// destroyAllResources destroys all resources defined in the profile.
func destroyAllResources(profile *parser.Profile, profilePath string) {
	if profile.LocalRegistry != nil {
		local_registry.DestroyLocalRegistry()
		profile.LocalRegistry = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Registry destroyed successfully")
	}

	if profile.Tunnel != nil {
		ngrok.DestroyTunnel()
		profile.Tunnel = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Tunnel destroyed successfully")
	}

	if profile.CloudResource != nil {
		azure.Destroy()
		profile.CloudResource = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Cloud resources destroyed successfully")
	}
}

// saveProfile saves the profile to the specified path
func saveProfile(profile *parser.Profile, profilePath string) {
	if err := parser.SaveProfile(profile, profilePath); err != nil {
		log.Printf("❌ Error saving profile: %v", err)
	}
}

// loadProfile loads the profile from the config file
func loadProfile() (*parser.Profile, string) {
	profilePath, err := parser.GetProfilePath()
	if err != nil {
		log.Printf("❌ Error getting profile path: %v", err)
		return nil, ""
	}

	profile, err := parser.LoadOrCreateProfile(profilePath)
	if err != nil {
		log.Printf("❌ Error loading or creating profile: %v", err)
		return nil, ""
	}
	return profile, profilePath
}
