package main

import (
	"os"
)

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
