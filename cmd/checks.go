package cmd

import (
	"fmt"
	"os"

	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CheckCmd = &cobra.Command{
	Use:   "checks",
	Short: "Run Service Checks",
	Long:  `Will get config file for checks and then running it at intervals`,
	Run:   startChecks,
}
var (
	checkcfg *viper.Viper
	cfgCheck string
	dryRun   bool
)

func init() {
	cobra.OnInitialize(initCheckConfig)
	CheckCmd.PersistentFlags().StringVar(&cfgCheck, "config", "", "service check config file (default is $HOME/.cyboard/checks.toml)")
	CheckCmd.Flags().BoolVarP(&dryRun, "dry", "", false, "Do a dry run of checks")
}

func initCheckConfig() {
	checkcfg = viper.New()
	if cfgCheck != "" {
		checkcfg.SetConfigFile(cfgCheck)
	}
	checkcfg.SetConfigName("checks")
	checkcfg.AddConfigPath("$HOME/.cyboard/")
	checkcfg.AddConfigPath(".")
	err := checkcfg.ReadInConfig()
	if err != nil {
		fmt.Println("Fatal error reading config file:", err)
		os.Exit(1)
	}
	server.SetupCfg(checkcfg, dryRun)
}

func startChecks(cmd *cobra.Command, args []string) {
	server.ChecksRun()
}
