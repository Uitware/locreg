package cmd

import (
	"fmt"
	"locreg/pkg/parser"
	"locreg/pkg/tunnels/ngrok"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "🌐 create a tunnel to expose registry to the public Internet",
	Long:  `🌐 create a tunnel to expose your local registry, protected with credentials, to the public Internet`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		config, err := parser.LoadConfig(configFilePath)
		if err != nil {
			fmt.Println(fmt.Errorf("failed to load config: %w", err))
			return
		}
		if config.Tunnel.Provider.Ngrok != (struct{}{}) {
			fmt.Println("❌ Please specify 'ngrok' in the config file. Or if you want to use another provider, " +
				"please wait for the next release or contribute by yourself 😺")
			return
		}
		ngrok.StartTunnel(configFilePath)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
