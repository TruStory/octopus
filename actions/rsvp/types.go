package main

import (
	"fmt"
	"os"

	"github.com/TruStory/octopus/services/truapi/truapi"
)

type UserJourneyResponse struct {
	Status int                        `json:"status"`
	Data   truapi.UserJourneyResponse `json:"data"`
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
