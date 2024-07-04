package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/local_registry"
	"log"
	"os"
	"os/exec"
)

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Run a local Docker registry",
	Long:  `This command runs a local Docker registry using Docker.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to get executable path: %v", err)
		}
		cmdTunnel := exec.Command(exePath, "tunnel")
		cmdTunnel.Stdout = os.Stdout
		cmdTunnel.Stderr = os.Stderr
		if err = cmdTunnel.Run(); err != nil {
			log.Fatalf("Failed to run tunnel: %v", err)
		}

		if err := local_registry.InitCommand(configFilePath); err != nil {
			fmt.Println("Error running registry:", err)
		}
	},
}

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate credentials of the local Docker registry",
	Long:  `This command rotates the credentials of the local Docker registry.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		err := local_registry.RotateCommand(configFilePath)
		if err != nil {
			fmt.Println("Error rotating registry credentials:", err)
		} else {
			fmt.Println("Credentials rotated successfully.")
		}
	},
}

func init() {
	registryCmd.AddCommand(rotateCmd)
	rootCmd.AddCommand(registryCmd)
}
