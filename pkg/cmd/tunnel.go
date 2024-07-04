package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/tunnels/ngrok"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "",
	Long:  ``,
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
