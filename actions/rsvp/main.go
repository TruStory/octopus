package main

import (
	"encoding/json"
	"fmt"
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

	userJourneyEndpoint := mustEnv("USER_JOURNEY_ENDPOINT")
	adminUsername := mustEnv("ADMIN_USERNAME")
	adminPassword := mustEnv("ADMIN_PASSWORD")
	inviteBatchSize, err := strconv.Atoi(mustEnv("INVITE_BATCH_SIZE"))
	if err != nil {
		log.Fatalln(err)
	}
	httpClient := &http.Client{
		Timeout: time.Second * 10,
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
		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?user_id=%d", userJourneyEndpoint, newUser.ID), nil)
		request.SetBasicAuth(adminUsername, adminPassword)
		if err != nil {
			log.Fatalln(err)
		}
		response, err := httpClient.Do(request)
		if err != nil {
			log.Fatalln(err)
		}
		defer response.Body.Close()

		var userJourney UserJourneyResponse
		err = json.NewDecoder(response.Body).Decode(&userJourney)
		if err != nil {
			log.Fatalln(err)
		}

		if !userHasBecomeEligible(userJourney) {
			fmt.Printf("not yet eligible. ❌\n")
			continue
		}

		// if yes...
		fmt.Printf("has become eligible. ✅\n")

		// give invites
		fmt.Printf("\tGranting %d new invites... ", inviteBatchSize)
		err = dbClient.GrantInvites(userJourney.Data.UserID, inviteBatchSize)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("✅\n")

		// check if they were referred by another user
		user, err := dbClient.UserByID(userJourney.Data.UserID)
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

		// TODO: reward TRU to the inviting user
		// Q: Should we check if the referrer users themselves are eligible yet or not?
		fmt.Printf("\tWas referred by %s. Rewarding them with TRU...", referrer.Username)
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

func userHasBecomeEligible(userJourney UserJourneyResponse) bool {
	for _, step := range requiredSteps {
		// if any step is not completed, the user is not eligible
		if !userJourney.Data.Steps[step] {
			return false
		}
	}

	return true
}
