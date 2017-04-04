package server

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   "cyboard",
	Short: "Scoring Engine",
	Long:  `This will start the Scoring Engine`,
	Run:   rootRun,
}

var CfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is $HOME/.cyboard/config.toml)")
}

func initConfig() {
	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	}
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.cyboard/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		Logger.Fatalf("Fatal error config file: %s\n", err)
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	fmt.Println(viper.GetString("appname"))
	fmt.Println(viper.GetString("server.ip"))
	fmt.Println(viper.GetString("server.http_port"))
	fmt.Println(viper.GetString("server.https_port"))
	fmt.Println(viper.GetString("server.cert"))
	fmt.Println(viper.GetString("server.key"))
	fmt.Println(viper.GetString("database.mongodb_uri"))
	err := cmd.Help()
	if err != nil {
		Logger.Println(err)
	}
}

func addCommands() {
	RootCmd.AddCommand(ServerCmd)
	RootCmd.AddCommand(CheckCmd)
}

func Execute() {
	addCommands()
	err := RootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
