package logger

import (
	"os"
	"strings"
)

// envBool Extract a string value from an environment variable
func envBool(key string, dflt bool) (setting bool) {
	s := os.Getenv(key)
	switch strings.ToLower(s) {
	default:
		setting = dflt
	case "f", "false", "off", "no", "0":
		setting = false
	case "t", "true", "on", "yes", "1":
		setting = true
	}
	return
}

// envString Extract a string value from an environment variable
func envString(key string, dflt string) (setting string) {
	setting = os.Getenv(key)
	if setting == "" {
		setting = dflt
	}
	return
}
