package cmd

import (
	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ServerCmd = &cobra.Command{
		Use:   "server",
		Short: "Web server for static pages and api",
		Long:  `This will run the web server`,
		Run:   serverRun,
	}
	serverConfig = viper.New()
)

func init() {
	flags := ServerCmd.Flags()
	flags.StringP("config", "c", "", "config file (default is $HOME/.cyboard/config.toml)")
	flags.Int("http-port", 8080, "HTTP Port for cyboard used for redirect")
	flags.Int("https-port", 8081, "HTTPS Port for cyboard")

	serverConfig.BindPFlag("configPath", flags.Lookup("config"))
	serverConfig.BindPFlag("server.http_port", flags.Lookup("http-port"))
	serverConfig.BindPFlag("server.https_port", flags.Lookup("https-port"))
}

func serverRun(cmd *cobra.Command, args []string) {
	// The server's config is `config.toml` by default
	initConfig(serverConfig, "config")
	checkForDebugDump(serverConfig)

	c := &server.Configuration{}
	mustUnmarshal(serverConfig, c)
	mustValidate(c)
	server.Run(c)
}
