package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	RunAddress        string
	DatabaseURI       string
	AccrualSystemAddr string
}

func NewConfig() *Config {
	return &Config{
		RunAddress:        getEnv("RUN_ADDRESS", ":8080"),
		DatabaseURI:       getEnv("DATABASE_URI", ""),
		AccrualSystemAddr: getEnv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8081"),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Trim(value, "\"'")
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(val)
		if err == nil {
			return v
		}
	}
	return defaultVal
}
