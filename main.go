package main

import (
	"fmt"

	"github.com/pereztr5/cyboard/web"
)

func main() {
	fmt.Printf("Starting Cyboard!\n")

	web.Start(":5443")
}
