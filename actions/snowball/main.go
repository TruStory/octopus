package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/octopus/services/truapi/truapi"
)

var allSteps = [...]db.UserJourneyStep{
	db.JourneyStepSignedUp,
	db.JourneyStepOneArgument,
	db.JourneyStepGivenOneAgree,
	db.JourneyStepReceiveFiveAgrees,
}

var requiredSteps = [...]db.UserJourneyStep{
	db.JourneyStepSignedUp,
	db.JourneyStepOneArgument,
	db.JourneyStepReceiveFiveAgrees,
}

var rewardForStep = map[db.UserJourneyStep]string{
	db.JourneyStepSignedUp:          mustEnv("REWARD_STEP_SIGNUP"),
	db.JourneyStepOneArgument:       mustEnv("REWARD_STEP_ONE_ARGUMENT"),
	db.JourneyStepReceiveFiveAgrees: mustEnv("REWARD_STEP_FIVE_AGREES"),
}

type usersflag []int64

func (u *usersflag) String() string {
	return fmt.Sprintf("%d", *u)
}

func (u *usersflag) Set(value string) error {
	tmp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		*u = append(*u, -1)
	} else {
		*u = append(*u, tmp)
	}
	return nil
}

func main() {
	var users usersflag
	flag.Var(&users, "u", "list of users for whom the service will run (optional)")
	flag.Parse()

	dbPort, err := strconv.Atoi(getEnv("PG_PORT", "5432"))
	if err != nil {
		log.Fatalln(err)
	}
	config := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_HOST", "localhost"),
			Port: dbPort,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	dbClient := db.NewDBClient(config)

	inviteBatchSize, err := strconv.Atoi(mustEnv("INVITE_BATCH_SIZE"))
	if err != nil {
		log.Fatalln(err)
	}

	var newUsers []db.User
	if len(users) == 0 {
		// get all the users who haven't completed their journey yet
		newUsers, err = getNewUsers(dbClient)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		newUsers, err = dbClient.UsersByID(users)
		if err != nil {
			log.Fatalln(err)
		}
	}
	fmt.Printf("Evaluating %d new user(s).\n", len(newUsers))

	for _, user := range newUsers {
		time.Sleep(1 * time.Second)
		fmt.Printf("Evaluating user with ID: %d -- ", user.ID)

		// checking for the progress made
		currentJourney, err := getCurrentJourney(user)
		if err != nil {
			log.Fatalln(err)
		}
		// if there are not new steps done by the user, we are done here
		if len(user.Meta.Journey) == len(currentJourney) {
			fmt.Printf("no progress made. ❗️(%s) vs (%s) \n", user.Meta.Journey, currentJourney)
			// moving on to the next one
			continue
		}
		additionalStepsCompleted := additionalStepsCompleted(currentJourney, user.Meta.Journey)

		// check if they are eligible to be given their first set of invites
		eligible := userHasBecomeEligible(user.Meta.Journey, currentJourney)
		if !eligible {
			// no invites to be given yet
			fmt.Printf("was already eligible or not yet eligible for invites. ❌\n")
		} else {
			// award the first set of invites
			fmt.Printf("has become eligible for invites. ✅\n")

			fmt.Printf("\tGranting %d new invites... ", inviteBatchSize)
			err = dbClient.GrantInvites(user.ID, inviteBatchSize)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("✅\n")
			sendNotification(app.RewardNotificationRequest{
				RewardeeID:   user.ID,
				RewardType:   app.RewardTypeInvite,
				RewardAmount: strconv.Itoa(inviteBatchSize),
			})
		}

		// if they were not referred by anyone, we are done for them
		if user.ReferredBy == 0 {
			// updating the current journey, and...
			err = dbClient.UpdateUserJourney(user.ID, currentJourney)
			if err != nil {
				log.Fatalln(err)
			}

			// moving on to the next one...
			continue
		}

		referrer, err := dbClient.UserByID(user.ReferredBy)
		if err != nil {
			log.Fatalln(err)
		}

		// if the user has become eligible, reward more invites to the inviting user
		if eligible {
			fmt.Printf("\tWas referred by %s. Granting them %d invites as well...", referrer.Username, inviteBatchSize)
			err = dbClient.GrantInvites(referrer.ID, inviteBatchSize)
			if err != nil {
				log.Fatalln(err)
			}
			sendNotification(app.RewardNotificationRequest{
				RewardeeID:   referrer.ID,
				RewardType:   app.RewardTypeInvite,
				RewardAmount: strconv.Itoa(inviteBatchSize),
				CauserID:     user.ID,
				CauserAction: app.RewardCauserActionJourneyComplete,
			})
			fmt.Printf("✅\n")
		}

		for _, step := range additionalStepsCompleted {
			reward, exists := rewardForStep[step]
			if !exists {
				// if no reward is present for this step,
				// move on to the next one...
				continue
			}
			fmt.Printf("\tRewarding referrer (%s) them with %s because %s is completed...", referrer.Username, reward, step)
			err = sendReward(*referrer, reward)
			if err != nil {
				log.Fatalln(err)
			}
			sendNotification(app.RewardNotificationRequest{
				RewardeeID:   referrer.ID,
				RewardType:   app.RewardTypeTru,
				RewardAmount: reward,
				CauserID:     user.ID,
				CauserAction: getCauserActionFromJourneyStep(step),
			})
			fmt.Printf("✅\n")
		}

		err = dbClient.UpdateUserJourney(user.ID, currentJourney)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func getNewUsers(dbClient *db.Client) ([]db.User, error) {
	users, err := dbClient.UsersWithIncompleteJourney()
	if err != nil {
		return users, err
	}

	return users, nil
}

func getCurrentJourney(user db.User) (journey []db.UserJourneyStep, err error) {
	response, err := makeHTTPRequest(http.MethodGet, fmt.Sprintf("%s?user_id=%d", mustEnv("ENDPOINT_USER_JOURNEY"), user.ID), nil)
	if err != nil {
		return
	}
	if response.StatusCode != 200 {
		return journey, fmt.Errorf("Fetching user journey for (%d) %s failed", user.ID, user.Username)
	}
	defer response.Body.Close()

	var userJourney UserJourneyResponse
	err = json.NewDecoder(response.Body).Decode(&userJourney)
	if err != nil {
		return
	}

	for _, step := range allSteps {
		if userJourney.Data.Steps[step] {
			journey = append(journey, step)
		}
	}

	return
}

func userHasBecomeEligible(previous, current []db.UserJourneyStep) bool {
	previouslyEligible := true
	currentlyEligble := true
	for _, step := range requiredSteps {
		// if any step is not completed, the user is not eligible
		if !containsStep(previous, step) {
			previouslyEligible = false
		}
	}

	for _, step := range requiredSteps {
		// if any step is not completed, the user is not eligible
		if !containsStep(current, step) {
			currentlyEligble = false
		}
	}

	// must not be already previously eligible, but become currently eligible
	return !previouslyEligible && currentlyEligble
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
	response, err := makeHTTPRequest(http.MethodPost, mustEnv("ENDPOINT_GIFT"), bodyBuffer)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("Gift payment to (%d) %s failed", user.ID, user.Username)
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

func containsStep(haystack []db.UserJourneyStep, needle db.UserJourneyStep) bool {
	for _, step := range haystack {
		if step == needle {
			return true
		}
	}

	return false
}

func additionalStepsCompleted(current []db.UserJourneyStep, previous []db.UserJourneyStep) []db.UserJourneyStep {
	var diff []db.UserJourneyStep

	for _, step := range current {
		if !containsStep(previous, step) {
			diff = append(diff, step)
		}
	}

	return diff
}

func getCauserActionFromJourneyStep(step db.UserJourneyStep) app.RewardCauserAction {
	switch step {
	case db.JourneyStepSignedUp:
		return app.RewardCauserActionSignedUp
	case db.JourneyStepOneArgument:
		return app.RewardCauserActionOneArgument
	case db.JourneyStepReceiveFiveAgrees:
		return app.RewardCauserActionReceiveFiveAgrees
	}

	return app.RewardCauserActionUnknown
}
