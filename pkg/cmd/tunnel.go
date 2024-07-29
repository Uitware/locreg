package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"locreg/pkg/parser"
	"locreg/pkg/tunnels/ngrok"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Create a tunnel to expose registry to the public Internet",
	Long:  `Create a tunnel to expose your local registry, protected with credentials, to the public Internet`,
	Run: func(cmd *cobra.Command, args []string) {
		configFilePath := "locreg.yaml"
		config, err := parser.LoadConfig(configFilePath)
		if err != nil {
			fmt.Println(fmt.Errorf("❌ failed to load config: %w", err))
			return
		}
		if !config.IsNgrokConfigured() {
			fmt.Println("❌ Please specify 'ngrok' in the config file. Or if you want to use another provider, " +
				"please wait for the next release or contribute by yourself")
			return
		}
		ngrok.RunNgrokTunnelContainer(config)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
