package cmd

import (
	"log"
	"os"

	"github.com/pereztr5/cyboard/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Web server for static pages and api",
	Long:  `This will run the web server`,
	Run:   serverRun,
}

// Logger is used to send logging messages to stdout.
var Logger = log.New(os.Stdout, " ", log.Ldate|log.Ltime|log.Lshortfile)

func init() {
	ServerCmd.Flags().Int("http_port", 8080, "HTTP Port for cyboard used for redirect")
	ServerCmd.Flags().Int("https_port", 1443, "HTTPS Port for cyboard")
	ServerCmd.PersistentFlags().String("mongodb_uri", "mongodb://127.0.0.1", "Address of MongoDB in use")
	viper.BindPFlag("server.http_port", ServerCmd.Flags().Lookup("http_port"))
	viper.BindPFlag("server.https_port", ServerCmd.Flags().Lookup("https_port"))
	viper.BindPFlag("database.mongodb_uri", ServerCmd.PersistentFlags().Lookup("mongodb_uri"))
}

func serverRun(cmd *cobra.Command, args []string) {
	server.Run()
}
