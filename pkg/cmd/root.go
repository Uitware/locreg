package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// TODO - refactor description

var rootCmd = &cobra.Command{
	Use:   "locreg",
	Short: "locreg is a CLI tool for managing deployments and tunnels",
	Long:  `locreg is a CLI tool for managing deployments and tunnels for various cloud providers.`,
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
