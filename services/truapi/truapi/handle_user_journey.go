package truapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

type UserJourneyStep string

const (
	JourneyStepOneArgument UserJourneyStep = "one_argument"
	JourneyStepFiveAgrees  UserJourneyStep = "five_agrees"
)

type UserJourneyResponse struct {
	UserID int64                    `json:"user_id"`
	Steps  map[UserJourneyStep]bool `json:"steps"`
}

// HandleUserJourney returns the progress of a user on the journey
func (ta *TruAPI) HandleUserJourney(w http.ResponseWriter, r *http.Request) {
	// only supports GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	inputUserID := r.FormValue("user_id")
	if inputUserID == "" {
		render.Error(w, r, "user id is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(inputUserID, 10, 64)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := ta.DBClient.UserByID(userID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if user == nil {
		render.Error(w, r, "user not found", http.StatusNotFound)
		return
	}

	steps := make(map[UserJourneyStep]bool)
	steps[JourneyStepOneArgument], err = isStepCompleted(ta, JourneyStepOneArgument, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[JourneyStepFiveAgrees], err = isStepCompleted(ta, JourneyStepFiveAgrees, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	response := UserJourneyResponse{
		UserID: userID,
		Steps:  steps,
	}

	render.Response(w, r, response, http.StatusOK)
}

func isStepCompleted(ta *TruAPI, step UserJourneyStep, user *db.User) (bool, error) {
	switch step {
	case JourneyStepOneArgument:
		return isOneArgumentStepComplete(ta, user)
	case JourneyStepFiveAgrees:
		return isFiveAgreesStepComplete(ta, user)
	}

	return false, errors.New("invalid journey step")
}

func isOneArgumentStepComplete(ta *TruAPI, user *db.User) (bool, error) {
	ctx := context.Background()
	arguments := ta.appAccountArgumentsResolver(ctx, queryByAddress{ID: user.Address})
	return len(arguments) > 1, nil
}

func isFiveAgreesStepComplete(ta *TruAPI, user *db.User) (bool, error) {
	ctx := context.Background()
	agrees := ta.agreesResolver(ctx, queryByAddress{ID: user.Address})
	return len(agrees) > 5, nil
}
