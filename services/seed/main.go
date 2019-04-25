package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	// seed a user
	apiEndpoint := mustEnv("SEED_API_ENDPOINT")

	// firing up the http client
	client := &http.Client{}

	// preparing the request
	request, err := http.NewRequest("POST", apiEndpoint+"/mock_register", nil)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	// processing the request
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	// reading the response
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	var mockedUser MockedRegisterResponse
	json.Unmarshal(responseBody, &mockedUser)
	fmt.Printf("Response: %v\n", mockedUser.Data.AuthenticationCookie)

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
