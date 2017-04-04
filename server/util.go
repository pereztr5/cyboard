package server

import "path/filepath"

// globRequired is the "Must" idiom for filepath.Glob
func mustGlobFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		Logger.Fatal(err)
	}
	return matches
}
