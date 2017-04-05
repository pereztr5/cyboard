package server

import (
	"path/filepath"
	"bufio"
	"os"
)

// globRequired is the "Must" idiom for filepath.Glob
func mustGlobFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		Logger.Fatal(err)
	}
	return matches
}

func ReadStdinLine() ([]byte, error) {
	stdin := bufio.NewScanner(os.Stdin)
	stdin.Scan()
	return stdin.Bytes(), stdin.Err()
}