package config

import (
	"os"
	"strconv"
	"strings"
)

func GetString(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func GetInt(key string, fallback int) int {
	value := GetString(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func GetFloat(key string, fallback float64) float64 {
	value := GetString(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func GetBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(GetString(key, "")))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func IntAtLeast(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func IntInRange(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func FloatAtLeast(value float64, minimum float64) float64 {
	if value < minimum {
		return minimum
	}
	return value
}

func FloatInRange(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
