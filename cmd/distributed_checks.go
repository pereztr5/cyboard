package cmd

import (
	"github.com/pereztr5/cyboard/server"
	"github.com/pereztr5/cyboard/server/monitor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	DistributedCheckCmd = &cobra.Command{
		Use:   "distributed-checks",
		Short: "Schedule and collect service checks against each teams' infrastructure",
		Run:   startDistributedChecks,
	}
)

func startDistributedChecks(cmd *cobra.Command, args []string) {
	c := new(server.Configuration)
	{
		checkConfig := viper.New()

		initConfig(checkConfig, "config")

		mustUnmarshal(checkConfig, c)
		mustValidate(c)
	}
	monitor.ChecksRun(c)
}
