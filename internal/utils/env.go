package utils

import (
	"os"
	"strconv"
)

func GetEnv(env, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

func GetEnvInt(env string, def int) int {
	if v := os.Getenv(env); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
