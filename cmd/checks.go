package cmd

import (
	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	CheckCmd = &cobra.Command{
		Use:   "checks",
		Short: "Run Service Checks",
		Long:  `Will get config file for checks and then running it at intervals`,
		Run:   startChecks,
	}
	checkConfig = viper.New()
)

func init() {
	flags := CheckCmd.Flags()
	flags.StringP("config", "c", "", "service check config file (default is $HOME/.cyboard/checks.toml)")
	//flags.BoolP("dry", "d", false, "Do a dry run of checks")

	checkConfig.BindPFlag("configPath", flags.Lookup("config"))
	//checkConfig.BindPFlag("dryRun", flags.Lookup("dry"))
}

func startChecks(cmd *cobra.Command, args []string) {
	// The service checker's config is `checks.toml` by default
	initConfig(checkConfig, "config")
	checkForDebugDump(checkConfig)

	c := &server.Configuration{}
	mustUnmarshal(checkConfig, c)
	// TODO: validate

	server.ChecksRun(c)
}
