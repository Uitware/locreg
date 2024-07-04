package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/parser"
	"locreg/pkg/tunnels/ngrok"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		config, err := parser.LoadConfig(configFilePath)
		if err != nil {
			fmt.Println(fmt.Errorf("failed to load config: %w", err))
			return
		}
		if config.Tunnel.Provider.Ngrok != (struct{}{}) {
			fmt.Println("Please specify 'ngrok' in the config file.")
			return
		}
		ngrok.StartTunnel(configFilePath)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
