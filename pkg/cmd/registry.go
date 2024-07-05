package cmd

import (
	"fmt"
	"locreg/pkg/local_registry"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "📍 run a local container registry",
	Long:  `📍 run a local registry, that is used for storing local development images and is exposed to public Internet via tunnel.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("❌ Failed to get executable path: %v", err)
		}
		cmdTunnel := exec.Command(exePath, "tunnel")
		cmdTunnel.Stdout = os.Stdout
		cmdTunnel.Stderr = os.Stderr
		if err = cmdTunnel.Run(); err != nil {
			log.Fatalf("❌ Failed to run tunnel: %v", err)
		}

		if err := local_registry.InitCommand(configFilePath); err != nil {
			fmt.Println("❌ Error running registry:", err)
		}
	},
}

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "🔄 rotate credentials of the local container registry",
	Long:  `🔄 rotates the credentials (username and password => token) of the local container registry.`,
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
