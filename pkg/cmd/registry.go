package cmd

import (
	"fmt"
	"github.com/Uitware/locreg/pkg/local_registry"
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/Uitware/locreg/pkg/tunnels/ngrok"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Run a local container registry",
	Long:  `Run a local registry, that is used for storing local development images and is exposed to public Internet via tunnel.`,
	Run: func(cmd *cobra.Command, args []string) {
		profile, _ := parser.LoadProfileData()
		if profile.Tunnel != nil {
			log.Fatalf("❌ Tunnel already exists. Please destroy it before creating a new one")
		}

		if profile.LocalRegistry != nil {
			log.Fatalf("❌ Local registry already exists. Please destroy it before creating a new one")

		}

		configFilePath := "locreg.yaml"
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("❌ Failed to get executable path: %v", err)
		}
		cmdTunnel := exec.Command(exePath, "tunnel")

		err = cmdTunnel.Run()
		if err != nil {
			log.Fatalf("❌ Failed to run tunnel: %v.\nCheck NGROK_AUTHTOKEN env variable value", err)
		}

		if err := local_registry.InitCommand(configFilePath); err != nil {
			err := ngrok.DestroyTunnel()
			if err != nil {
				log.Fatalf("❌ error destroying tunnel: %v. \nYou need to do this manualy", err)
			}
			log.Fatalf("❌ error running registry: %v", err)
		}
	},
}

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate credentials of the local container registry",
	Long:  `Rotates the credentials (username and password => token) of the local container registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		err := local_registry.RotateCommand(configFilePath)
		if err != nil {
			fmt.Println("❌ Error rotating registry credentials:", err)
		} else {
			fmt.Println("✅ Credentials rotated successfully.")
		}
	},
}

func init() {
	registryCmd.AddCommand(rotateCmd)
	rootCmd.AddCommand(registryCmd)
}
