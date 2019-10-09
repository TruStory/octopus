package truapi

import (
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// HandleRequestTru handles requests for comments
func (ta *TruAPI) HandleRequestTru(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ta.handleRequestTru(w, r)
	default:
		render.Error(w, r, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ta *TruAPI) handleRequestTru(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		render.Error(w, r, Err401NotAuthenticated.Error(), http.StatusUnauthorized)
		return
	}

	// dispatch a slack message here
	userProfile := ta.userProfileResolver(ctx, user.Address)
	if userProfile == nil {
		render.Error(w, r, Err422UnprocessableEntity.Error(), http.StatusUnprocessableEntity)
		return
	}

	ta.sendRequestTruToSlack(user.Address, *userProfile)

	render.JSON(w, r, nil, http.StatusOK)
}
