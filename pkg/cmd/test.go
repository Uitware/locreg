package cmd

import (
	"github.com/Uitware/locreg/pkg/parser"
	"github.com/spf13/cobra"
	"log"
)

var testCMD = &cobra.Command{
	Use:   "test",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := parser.LoadConfig("locreg.yaml")
		if err != nil {
			return
		}
		log.Print(config)
	},
}

func init() {
	rootCmd.AddCommand(testCMD)
}
