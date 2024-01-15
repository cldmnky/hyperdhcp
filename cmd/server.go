package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cldmnky/hyperdhcp/internal/dhcp"
)

var serverCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		// get config flag
		// pass to dhcp.Run
		//
		cfg, err := rootCmd.PersistentFlags().GetString("config")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Printf("Config: %v\n", cfg)
		fmt.Println("Viper config:", viper.GetString("foo"))
		dhcp.Run(dhcp.NewConfig(viper.ConfigFileUsed()))
	},
}
