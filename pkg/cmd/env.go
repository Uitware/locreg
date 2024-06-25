package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// TODO - refactor description

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment values used by locreg tool",
	Long:  `Manage environment values used by locreg tool, such as viewing and editing the current environment configuration.`,
}

var envEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit current environment values used by locreg tool",
	Run: func(cmd *cobra.Command, args []string) {
		// Placeholder for actual editing logic
		fmt.Println("Editing current environment values...")
	},
}

var envShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current environment values used by locreg tool",
	Run: func(cmd *cobra.Command, args []string) {
		// Placeholder for actual showing logic
		envVars := os.Environ()
		fmt.Println("Current environment values:")
		for _, env := range envVars {
			fmt.Println(env)
		}
	},
}

func init() {
	envCmd.AddCommand(envEditCmd)
	envCmd.AddCommand(envShowCmd)
	rootCmd.AddCommand(envCmd)
}
