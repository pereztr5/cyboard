package server

import (
	"bufio"
	"os"
	"path/filepath"
)

// globRequired is the "Must" idiom for filepath.Glob
func mustGlobFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if matches == nil {
		panic("Unable to locate required template files in working dir that match: " + pattern)
	} else if err != nil {
		panic(err) // Programmer error, bad glob pattern
	}
	return matches
}

func ReadStdinLine() ([]byte, error) {
	stdin := bufio.NewScanner(os.Stdin)
	stdin.Scan()
	return stdin.Bytes(), stdin.Err()
}

func Int64Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
