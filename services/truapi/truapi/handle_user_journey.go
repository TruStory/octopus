package truapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

type UserJourneyResponse struct {
	UserID int64                       `json:"user_id"`
	Steps  map[db.UserJourneyStep]bool `json:"steps"`
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

	steps := make(map[db.UserJourneyStep]bool)
	steps[db.JourneyStepSignedUp], err = isStepCompleted(ta, db.JourneyStepSignedUp, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[db.JourneyStepOneArgument], err = isStepCompleted(ta, db.JourneyStepOneArgument, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[db.JourneyStepFiveAgrees], err = isStepCompleted(ta, db.JourneyStepFiveAgrees, user)
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

func isStepCompleted(ta *TruAPI, step db.UserJourneyStep, user *db.User) (bool, error) {
	switch step {
	case db.JourneyStepSignedUp:
		return user.Address != "", nil
	case db.JourneyStepOneArgument:
		return isOneArgumentStepComplete(ta, user)
	case db.JourneyStepFiveAgrees:
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
