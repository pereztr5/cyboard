package cmd

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
	Run:   RootRun,
}

var CfgFile string

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is $HOME/.cyboard/config.toml)")
	RootCmd.AddCommand(ServerCmd)
	RootCmd.AddCommand(CheckCmd)
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
		fmt.Println("Fatal error config file:", err)
		os.Exit(1)
	}
}

func RootRun(cmd *cobra.Command, args []string) {
	// For debugging purposes
	fmt.Println(viper.GetString("appname"))
	fmt.Println(viper.GetString("server.ip"))
	fmt.Println(viper.GetString("server.http_port"))
	fmt.Println(viper.GetString("server.https_port"))
	fmt.Println(viper.GetString("server.cert"))
	fmt.Println(viper.GetString("server.key"))
	fmt.Println(viper.GetString("database.mongodb_uri"))
	err := cmd.Help()
	if err != nil {
		fmt.Println(err)
	}
}
