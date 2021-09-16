package util

import (
	"os"
	"strconv"
	"strings"
)

func GetEnvString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	data, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvFloat64(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	data, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	value = strings.ToLower(value)
	switch value {
	case "true":
		return true
	case "false":
		return false
	default:
		return defaultValue
	}
}
