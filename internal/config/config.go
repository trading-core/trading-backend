// Package config Simple code to get values from the environment.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/kduong/trading-backend/internal/logger"
)

// EnvStringOrFatal -- fetch a string from the environment, or Fatal if it doesn't exist.
func EnvStringOrFatal(key string) (value string) {
	value = os.Getenv(key)
	if value == "" {
		logger.Fatalf("The following environment variable must be set: %s", key)
	}
	return
}

// EnvInt Extract an integer value from an environment variable
func EnvInt(key string, dflt int) int {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		logger.Fatalf("Value from environment is not an integer: key %s value %s", key, s)
	}
	return i
}

// EnvInt32 Extract an integer value from an environment variable
func EnvInt32(key string, dflt int32) int32 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		logger.Fatalf("Value from environment is not a 32-bit integer: key %s value %s", key, s)
	}
	return int32(i)
}

// EnvInt64 Extract an integer value from an environment variable
func EnvInt64(key string, dflt int64) int64 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		logger.Fatalf("Value from environment is not a 64-bit integer: key %s value %s", key, s)
	}
	return i
}

// EnvUint Extract an unsigned integer value from an environment variable
func EnvUint(key string, dflt uint) uint {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		logger.Fatalf("Value from environment is not an unsigned integer: key %s value %s", key, s)
	}
	return uint(i)
}

// EnvUint32 Extract an integer value from an environment variable
func EnvUint32(key string, dflt uint32) uint32 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		logger.Fatalf("Value from environment is not an unsigned 32-bit integer: key %s value %s", key, s)
	}
	return uint32(i)
}

// EnvUint64 Extract an integer value from an environment variable
func EnvUint64(key string, dflt uint64) uint64 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		logger.Fatalf("Value from environment is not an unsigned 64-bit integer: key %s value %s", key, s)
	}
	return i
}

// EnvString Extract a string value from an environment variable
func EnvString(key string, dflt string) (setting string) {
	setting = os.Getenv(key)
	if setting == "" {
		setting = dflt
	}
	return
}

// EnvBool Extract a string value from an environment variable
func EnvBool(key string, dflt bool) (setting bool) {
	s := os.Getenv(key)
	setting, err := strconv.ParseBool(s)
	if err != nil {
		setting = dflt
	}
	return
}

// EnvDuration Extract a duration from an environment variable
func EnvDuration(key string, dflt time.Duration) (setting time.Duration) {
	s := os.Getenv(key)
	setting, err := time.ParseDuration(s)
	if err != nil {
		setting = dflt
	}
	return setting
}

// EnvFloat32 Extract a Float32 value from an environment variable
func EnvFloat32(key string, dflt float32) float32 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseFloat(s, 32)
	if err != nil {
		logger.Fatalf("Value from environment is not a 32-bit floating point number: key %s value %s", key, s)
	}
	return float32(i)
}

// EnvFloat64 Extract a Float64 value from an environment variable
func EnvFloat64(key string, dflt float64) float64 {
	s := os.Getenv(key)
	if s == "" {
		return dflt
	}
	i, err := strconv.ParseFloat(s, 64)
	if err != nil {
		logger.Fatalf("Value from environment is not a 64-bit floating point number: key %s value %s", key, s)
	}
	return i
}
