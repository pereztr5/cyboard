package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/pereztr5/cyboard/server"
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
	flags.StringP("config", "c", "", "config file (default ./config.toml)")
	flags.BoolVar(&dumpConfig, "dump-config", false,
		"Prints the parsed server / check config (for debugging purposes)")
	flags.String("postgres-uri", "postgresql://cybot@localhost/cyboard",
		"Connection string for PostgreSQL. Also configured with the environment var: `CY_POSTGRES_URI`")
	flags.BoolP("stdout", "s", false, "Log to standard out")

	RootCmd.AddCommand(ServerCmd, CheckCmd)
}

// initConfig loads the config file from disk, searching in order:
// `-c|--config` option path, configName in the cwd, or configName in `$HOME/.cyboard/`
//
// This function is used for both the CTF & Service checker (but not the RootCmd)
func initConfig(v *viper.Viper, configName string) {
	// Bind global flags & env vars to this specific config's values
	flags := RootCmd.PersistentFlags()
	v.BindPFlag("configPath", flags.Lookup("config"))
	v.BindPFlag("database.postgres_uri", flags.Lookup("postgres-uri"))
	v.BindPFlag("log.stdout", flags.Lookup("stdout"))

	// Env var may be used for sensitive connection strings (alternative to .pgpass file)
	v.BindEnv("database.postgres_uri", "CY_POSTRES_URI")

	// Set defaults
	v.SetDefault("server.rate_limit", true)
	v.SetDefault("server.ctf_file_dir", "data/ctf")
	v.SetDefault("service_monitor.checks_dir", "data/scripts")

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

	checkForDebugDump(v)
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

func mustValidate(cfg *server.Configuration) {
	if err := cfg.Validate(); err != nil {
		fmt.Println("Config file validation failed:", err)
		os.Exit(1)
	}
}

func RootRun(cmd *cobra.Command, args []string) {
	cmd.Help()
}
