package cmd

import (
	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	CheckCmd = &cobra.Command{
		Use:   "checks",
		Short: "Monitor and score each team's infrastructure",
		Run:   startChecks,
	}
)

func startChecks(cmd *cobra.Command, args []string) {
	c := new(server.Configuration)
	{
		checkConfig := viper.New()

		initConfig(checkConfig, "config")

		mustUnmarshal(checkConfig, c)
		mustValidate(c)
	}
	server.ChecksRun(c)
}
