package loader

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpandEnv performs ${VAR} expansion on environment variables
// Returns error if any variable is missing or if HADES_* is defined
func (l *Loader) ExpandEnv(env map[string]string) (map[string]string, error) {
	// Check for HADES_* user overrides
	for key := range env {
		if strings.HasPrefix(key, "HADES_") {
			return nil, fmt.Errorf("user cannot define HADES_* environment variables: %s", key)
		}
	}

	expanded := make(map[string]string)
	for key, value := range env {
		expandedValue, err := expandString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to expand env var %s: %w", key, err)
		}
		expanded[key] = expandedValue
	}

	return expanded, nil
}

// expandString expands ${VAR} references in a string
func expandString(s string) (string, error) {
	var missingVars []string

	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := match[2 : len(match)-1]

		value, ok := os.LookupEnv(varName)
		if !ok {
			missingVars = append(missingVars, varName)
			return match
		}
		return value
	})

	if len(missingVars) > 0 {
		return "", fmt.Errorf("missing OS environment variables: %s", strings.Join(missingVars, ", "))
	}

	return result, nil
}
