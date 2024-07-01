package cmd

import (
	"fmt"

	"locreg/pkg/tunnels/cloudflared"
	"locreg/pkg/tunnels/microsoft_dev_tunnels"
	"locreg/pkg/tunnels/ngrok"

	"github.com/spf13/cobra"
)

var startTunnel = &cobra.Command{
	Use:   "tunnel [tunnel_provider]",
	Short: "Start a tunnel with a specified provider",
	Long:  "Start a tunnel with a specified provider to expose your Docker registry.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]
		switch provider {
		case "cf":
			cloudflared.StartTunnel()
		case "ngrok":
			ngrok.StartTunnel()
		case "mstunnel":
			microsoft_dev_tunnels.StartTunnel()
		default:
			fmt.Println("Unknown provider:", provider)
		}
	},
}

func init() {
	rootCmd.AddCommand(startTunnel)
}
