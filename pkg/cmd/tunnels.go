package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/tunnels/ngrok"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "📝 Create ngrok tunnel to local registry",
	Long:  `Create ngrok tunnel to local registry to expose it to the internet. For later use as registry in cloud deployment.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "config.yaml"
		err := ngrok.StartTunnel(configFilePath)
		if err != nil {
			fmt.Println("Error running tunnel:", err)
		} else {
			fmt.Println("tunnel is running.")
		}
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
