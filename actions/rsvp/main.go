package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
)

var requiredSteps = [...]truapi.UserJourneyStep{
	truapi.JourneyStepOneArgument,
	truapi.JourneyStepFiveAgrees,
}

func main() {
	config := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_ADDR", "localhost"),
			Port: 5432,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}

	inviteBatchSize, err := strconv.Atoi(mustEnv("INVITE_BATCH_SIZE"))
	if err != nil {
		log.Fatalln(err)
	}
	dbClient := db.NewDBClient(config)

	// get all the users who haven't been given out first batch of invites
	newUsers, err := getNewUsers(dbClient)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Evaluating %d new users.\n", len(newUsers))

	for _, newUser := range newUsers {
		fmt.Printf("Evaluating user with ID: %d -- ", newUser.ID)

		// check if they are eligible to be given their first set of invites
		eligible, err := userHasBecomeEligible(newUser)
		if err != nil {
			log.Fatal(err)
		}
		if !eligible {
			fmt.Printf("not yet eligible. ❌\n")
			continue
		}

		// if yes...
		fmt.Printf("has become eligible. ✅\n")

		// give invites
		fmt.Printf("\tGranting %d new invites... ", inviteBatchSize)
		err = dbClient.GrantInvites(newUser.ID, inviteBatchSize)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("✅\n")

		// check if they were referred by another user
		user, err := dbClient.UserByID(newUser.ID)
		if err != nil {
			log.Fatalln(err)
		}

		if user.ReferredBy == 0 {
			continue
		}

		// if yes...
		referrer, err := dbClient.UserByID(user.ReferredBy)
		if err != nil {
			log.Fatalln(err)
		}
		// reward more invites to the inviting user
		fmt.Printf("\tWas referred by %s. Granting them %d invites as well...", referrer.Username, inviteBatchSize)
		err = dbClient.GrantInvites(referrer.ID, inviteBatchSize)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("✅\n")

		// Q: Should we check if the referrer users themselves are eligible yet or not?
		fmt.Printf("\tWas referred by %s. Rewarding them with TRU...", referrer.Username)
		err = sendReward(*referrer, mustEnv("REWARD_STEP_FIVE_AGREES"))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("✅\n")
	}
}

func getNewUsers(dbClient *db.Client) ([]db.User, error) {
	users := make([]db.User, 0)
	err := dbClient.Model(&users).
		Where("invites_valid_until IS NULL"). // meaning, they have never been given any invites
		Select()

	if err != nil {
		return users, err
	}

	return users, nil
}

func userHasBecomeEligible(user db.User) (bool, error) {
	response, err := makeHTTPRequest(http.MethodGet, fmt.Sprintf("%s?user_id=%d", mustEnv("ENDPOINT_USER_JOURNEY"), user.ID), nil)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	var userJourney UserJourneyResponse
	err = json.NewDecoder(response.Body).Decode(&userJourney)
	if err != nil {
		return false, err
	}

	for _, step := range requiredSteps {
		// if any step is not completed, the user is not eligible
		if !userJourney.Data.Steps[step] {
			return false, err
		}
	}

	return true, nil
}

func sendReward(user db.User, amount string) error {
	body := struct {
		UserID int64  `json:"user_id"`
		Amount string `json:"amount"`
	}{
		UserID: user.ID,
		Amount: amount,
	}
	bodyBuffer := new(bytes.Buffer)
	err := json.NewEncoder(bodyBuffer).Encode(body)
	if err != nil {
		return err
	}
	_, err = makeHTTPRequest(http.MethodPost, mustEnv("ENDPOINT_GIFT"), bodyBuffer)
	if err != nil {
		return err
	}

	return nil
}

func makeHTTPRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	request, err := http.NewRequest(method, endpoint, body)
	request.SetBasicAuth(mustEnv("ADMIN_USERNAME"), mustEnv("ADMIN_PASSWORD"))
	if err != nil {
		log.Fatalln(err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
