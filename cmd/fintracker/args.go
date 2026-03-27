package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fintracker/internal/tui"
)

func parseArgs(args []string) ([]tui.ImportSpec, error) {
	var specs []tui.ImportSpec

	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected account:path, got %q", arg)
		}

		account := parts[0]
		path := expandHome(parts[1])

		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("file %q: %w", path, err)
		}

		specs = append(specs, tui.ImportSpec{
			Path:    path,
			Account: account,
		})

	}

	return specs, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
