package main

//Start testing service checker

import (
	"fmt"
	"os"

	"github.com/pereztr5/cyboard/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
