package cmd

import (
	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ServerCmd = &cobra.Command{
		Use:   "server",
		Short: "Web application & API for scoreboard",
		Run:   serverRun,
	}
)

func init() {
	flags := ServerCmd.Flags()
	flags.Int("http-port", 8080, "HTTP Port for cyboard used for redirect")
	flags.Int("https-port", 8081, "HTTPS Port for cyboard")
}

func serverRun(cmd *cobra.Command, args []string) {
	c := new(server.Configuration)
	{
		serverConfig := viper.New()
		serverConfig.BindPFlag("server.http_port", cmd.Flags().Lookup("http-port"))
		serverConfig.BindPFlag("server.https_port", cmd.Flags().Lookup("https-port"))

		initConfig(serverConfig, "config")

		mustUnmarshal(serverConfig, c)
		mustValidate(c)
	}
	server.Run(c)
}
