package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   "cyboard",
	Short: "Scoring Engine",
	Long:  `Root of the Scoring Engine application (see Available Commands)`,
	Run:   RootRun,
}
var (
	dumpConfig    bool
	rootCmdConfig = viper.New()
)

func init() {
	// This declares top level flags, shared by all commands
	flags := RootCmd.PersistentFlags()
	flags.BoolVar(&dumpConfig, "dump-config", false,
		"Prints the parsed server / check config (for debugging purposes)")
	flags.String("mongodb-uri", "mongodb://127.0.0.1",
		"Address of MongoDB instance to use. Also configured with the environment var: `MONGODB_URI`")
	flags.BoolP("stdout", "s", false, "Log to standard out")

	RootCmd.AddCommand(ServerCmd, CheckCmd)
}

// initConfig loads the config file from disk, searching in order:
// `-c|--config` option path, configName in the cwd, or configName in `$HOME/.cyboard/`
//
// This function is used for both the CTF & Service checker (but not the RootCmd)
func initConfig(v *viper.Viper, configName string) {
	path := v.GetString("configPath")
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(configName)
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.cyboard/")
	}

	if err := v.ReadInConfig(); err != nil {
		fmt.Println("Fatal error config file:", err)
		os.Exit(1)
	}

	// Bind global flags to this specific config's values
	flags := RootCmd.PersistentFlags()
	v.BindPFlag("database.mongodb_uri", flags.Lookup("mongodb-uri"))
	v.BindPFlag("log.stdout", flags.Lookup("stdout"))
}

// checkForDebugDump will dump the parsed config file and then exit the app
func checkForDebugDump(v *viper.Viper) {
	if dumpConfig {
		// v.ConfigFileUsed is empty when viper is given an explicit config path with `v.SetConfigFile()`
		fileUsed := v.ConfigFileUsed()
		if fileUsed != "" {
			fmt.Println(fileUsed)
		}

		keys := v.AllKeys()
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %s: %v\n", k, v.Get(k))
		}
		os.Exit(0)
	}
}

// mustUnmarshal unmarshals a viper config into the cfg struct, or errors the app and exits
func mustUnmarshal(v *viper.Viper, cfg interface{}) {
	if err := v.Unmarshal(cfg); err != nil {
		fmt.Println("Unmarshal of config failed:", err)
		os.Exit(1)
	}
}

func RootRun(cmd *cobra.Command, args []string) {
	cmd.Help()
}
