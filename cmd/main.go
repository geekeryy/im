package main

import (
	"im/server/discovery"
	"im/server/imgateway"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "im",
	Short: "im",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	rootCmd.AddCommand(discoveryCmd)
	rootCmd.AddCommand(imGatewayCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "start discovery server with args[0]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			os.Setenv("IM_DISCOVERY_ADDR", args[0])
		}
		discovery.Run()
	},
}

var imGatewayCmd = &cobra.Command{
	Use:   "imgateway",
	Short: "start im gateway server with args[0]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			os.Setenv("IM_GATEWAY_ADDR", args[0])
		}
		imgateway.Run()
	},
}
