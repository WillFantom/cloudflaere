package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "dump configuration to the log and exit",
		Run: func(cmd *cobra.Command, args []string) {
			logrus.WithFields(viper.AllSettings()).Infoln("configuration")
		},
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
}
