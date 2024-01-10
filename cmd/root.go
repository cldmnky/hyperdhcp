package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "hyperdhcp",
		Short: "A DHCP server for Hypershift",
	}
)

func Execute() error {

	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".hyperdhcp.yaml", "config file (default is .hyperdhcp.yaml)")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	zapfs := flag.NewFlagSet("zap", flag.ExitOnError)
	opts := &zap.Options{}
	opts.BindFlags(zapfs)
	opts.BindFlags(flag.CommandLine)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().AddGoFlagSet(zapfs)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(opts)))
	rootCmd.AddCommand(managerCmd)
	rootCmd.AddCommand(serverCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".hyperdhcp")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}
