package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// TODO - refactor description

var rootCmd = &cobra.Command{
	Use:   "locreg",
	Short: "🚀☁️ locreg enables **registryless** approach for serverless applications deployment",
	Long:  `🚀☁️ locreg is a CLI tool for managing deployments for various cloud providers, using a local container registry and a tunnel.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()
	rootCmd.Root().CompletionOptions.DisableDefaultCmd = true
}
