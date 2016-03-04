package main

import (
	"fmt"

	"github.com/gorilla/securecookie"
)

func main() {
	hashkey := securecookie.GenerateRandomKey(64)
	blockkey := securecookie.GenerateRandomKey(32)
	fmt.Printf("hashkey = \"%x\"\nblockkey = \"%x\"\n", hashkey, blockkey)
}
