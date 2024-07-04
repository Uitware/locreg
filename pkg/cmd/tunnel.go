package cmd

import (
	"github.com/spf13/cobra"
	"locreg/pkg/tunnels/ngrok"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "config.yaml"
		ngrok.StartTunnel(configFilePath)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
