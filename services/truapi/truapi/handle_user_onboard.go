package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UserOnboardRequest represents the JSON request for updating onboarding flow
type UserOnboardRequest struct {
	OnboardFollowCommunities *bool `json:"onboard_follow_communities,omitempty"`
	OnboardCarousel          *bool `json:"onboard_carousel,omitempty"`
	OnboardContextual        *bool `json:"onboard_contextual,omitempty"`
}

// HandleUserOnboard takes a `UserOnboardStepRequest` and returns a 200 response
func (ta *TruAPI) HandleUserOnboard(w http.ResponseWriter, r *http.Request) {
	// only support POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request := &UserOnboardRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	meta := &db.UserMeta{
		OnboardFollowCommunities: request.OnboardFollowCommunities,
		OnboardCarousel:          request.OnboardCarousel,
		OnboardContextual:        request.OnboardContextual,
	}
	err = ta.DBClient.SetUserMeta(user.ID, meta)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}
