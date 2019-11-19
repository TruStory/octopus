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
	steps[db.JourneyStepSignedUp], err = ta.isStepCompleted(r.Context(), db.JourneyStepSignedUp, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[db.JourneyStepOneArgument], err = ta.isStepCompleted(r.Context(), db.JourneyStepOneArgument, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[db.JourneyStepReceiveFiveAgrees], err = ta.isStepCompleted(r.Context(), db.JourneyStepReceiveFiveAgrees, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	steps[db.JourneyStepGivenOneAgree], err = ta.isStepCompleted(r.Context(), db.JourneyStepGivenOneAgree, user)
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

func (ta *TruAPI) isStepCompleted(ctx context.Context, step db.UserJourneyStep, user *db.User) (bool, error) {
	switch step {
	case db.JourneyStepSignedUp:
		return ta.isSignedUpStepComplete(ctx, user), nil
	case db.JourneyStepOneArgument:
		return ta.isOneArgumentStepComplete(ctx, user), nil
	case db.JourneyStepReceiveFiveAgrees:
		return ta.isFiveAgreesStepComplete(ctx, user), nil
	case db.JourneyStepGivenOneAgree:
		return ta.isGivenOneAgreeStepComplete(ctx, user), nil
	}

	return false, errors.New("invalid journey step")
}

func (ta *TruAPI) isSignedUpStepComplete(ctx context.Context, user *db.User) bool {
	// if an address is assigned to the user, it means,
	// the user is active, i.e. verified email if signed up via email,
	// or signed up via twitter
	return user.Address != ""
}

func (ta *TruAPI) isOneArgumentStepComplete(ctx context.Context, user *db.User) bool {
	if user.Address == "" {
		return false
	}
	arguments := ta.appAccountArgumentsResolver(ctx, queryByAddress{ID: user.Address})
	return len(arguments) >= 1
}

func (ta *TruAPI) isFiveAgreesStepComplete(ctx context.Context, user *db.User) bool {
	if user.Address == "" {
		return false
	}

	arguments := ta.appAccountArgumentsResolver(ctx, queryByAddress{ID: user.Address})

	agreesReceived := 0
	for _, argument := range arguments {
		agreesReceived += argument.UpvotedCount

		// bailing out of the loop, as soon as we get the counter to 5
		if agreesReceived >= 5 {
			return true
		}
	}

	return false
}

func (ta *TruAPI) isGivenOneAgreeStepComplete(ctx context.Context, user *db.User) bool {
	if user.Address == "" {
		return false
	}

	agrees := ta.agreesResolver(ctx, queryByAddress{ID: user.Address})

	return len(agrees) >= 1
}
