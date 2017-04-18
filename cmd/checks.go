package cmd

import (
	"github.com/fsnotify/fsnotify"
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
		Logger.Fatal("Fatal error reading config file:", err)
	}
	// FIXME(butters): There's an unfortunate race condition in the Viper library.
	//        https://github.com/spf13/viper/issues/174
	// The gist is that there's not synchronization mechanism for this
	// feature, so, if the config gets updated really quickly, the check
	// service would collapse. We could just copy the WatchConfig code
	// and add our own shared file lock as a quick patch.
	var cfgNeedsReload bool
	checkcfg.WatchConfig()
	checkcfg.OnConfigChange(func(in fsnotify.Event) {
		cfgNeedsReload = true
		Logger.Println(checkcfg.ConfigFileUsed(), "has been updated. ")
		Logger.Println("Settings will reload live at the next set of checks.")
	})
	server.SetupCfg(checkcfg, dryRun, cfgNeedsReload)
}

func startChecks(cmd *cobra.Command, args []string) {
	server.ChecksRun()
}
