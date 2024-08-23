package cmd

import (
	"fmt"
	"github.com/Uitware/locreg/pkg/local_registry"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/Uitware/locreg/pkg/providers/aws"
	"github.com/Uitware/locreg/pkg/providers/azure"
	"github.com/Uitware/locreg/pkg/tunnels/ngrok"
	"github.com/spf13/cobra"
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

		profile, profilePath := parser.LoadProfileData()
		if profile == nil {
			log.Fatalf("❌ Failed to load profile.")
		}

		switch resource {
		case "registry":
			if profile.LocalRegistry != nil {
				err := local_registry.DestroyLocalRegistry()
				if err != nil {
					log.Fatalf("❌ Error destroying local registry: %v", err)
				}
				profile.LocalRegistry = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Registry destroyed successfully")
			}

		case "tunnel":
			if profile.Tunnel != nil {
				err := ngrok.DestroyTunnel()
				if err != nil {
					log.Fatalf("❌ Error destroying tunnel: %v", err)
				}
				profile.Tunnel = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Tunnel destroyed successfully")
			}

		case "cloud":
			if profile.AzureCloudResource != nil {
				azure.Destroy()
				profile.AzureCloudResource = nil
				saveProfile(profile, profilePath)
				fmt.Println("✅ Cloud resources destroyed successfully")
			}
			if profile.AWSCloudResource != nil {
				aws.Destroy()
				profile.AWSCloudResource = nil
				profile.Save()
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
		err := local_registry.DestroyLocalRegistry()
		if err != nil {
			log.Fatalf("❌ Error destroying local registry: %v", err)
		}
		profile.LocalRegistry = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Registry destroyed successfully")
	}

	if profile.Tunnel != nil {
		err := ngrok.DestroyTunnel()
		if err != nil {
			log.Fatalf("❌ Error destroying tunnel: %v", err)
		}
		profile.Tunnel = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Tunnel destroyed successfully")
	}

	if profile.AzureCloudResource != nil {
		azure.Destroy()
		profile.AzureCloudResource = nil
		saveProfile(profile, profilePath)
		fmt.Println("✅ Cloud resources destroyed successfully")
	}
}

// saveProfile saves the profile
func saveProfile(profile *parser.Profile, profilePath string) {
	if err := parser.SaveProfile(profile, profilePath); err != nil {
		log.Printf("❌ Error saving profile: %v", err)
	}
}
