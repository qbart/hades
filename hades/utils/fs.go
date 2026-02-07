package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func FileHasValidExtension(path string) bool {
	return strings.HasSuffix(path, ".hades.yml") || strings.HasSuffix(path, ".hades.yaml")
}

// ExpandPath expands ~ to the user's home directory and resolves relative paths
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		return os.UserHomeDir()
	}

	// Make absolute if relative
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	return path, nil
}
