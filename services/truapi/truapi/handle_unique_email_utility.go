package truapi

import (
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UniqueEmailResponse represents the http response to check uniqueness of a username
type UniqueEmailResponse struct {
	IsUnique    bool `json:"is_unique"`
	IsValidated bool `json:"is_validated"`
}

// HandleUniqueEmailUtility checks if the provided username is unique or not
func (ta *TruAPI) HandleUniqueEmailUtility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		render.Error(w, r, "method not method", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")

	user, err := ta.DBClient.UserByEmail(email)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	response := &UniqueEmailResponse{
		IsUnique: false,
	}

	if user == nil {
		response.IsUnique = true
	} else {
		userVerified, err := ta.DBClient.VerifiedUserByID((*user).ID)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
		}
		response.IsValidated = userVerified != nil
	}

	render.Response(w, r, response, http.StatusOK)
}
