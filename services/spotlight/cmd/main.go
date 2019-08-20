package main

import (
	"fmt"
	"os"

	"github.com/TruStory/octopus/services/spotlight"
)

func main() {
	service := spotlight.NewService(getEnv("PORT", "54448"), mustEnv("SPOTLIGHT_GRAPHQL_ENDPOINT"))

	service.Run()
}
func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}
