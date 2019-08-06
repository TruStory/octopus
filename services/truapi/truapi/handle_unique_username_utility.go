package truapi

import (
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UniqueUsernameResponse represents the http response to check uniqueness of a username
type UniqueUsernameResponse struct {
	Username string `json:"username"`
	IsUnique bool   `json:"is_unique"`
}

// HandleUniqueUsernameUtility checks if the provided username is unique or not
func (ta *TruAPI) HandleUniqueUsernameUtility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		render.Error(w, r, "method not method", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")

	user, err := ta.DBClient.UserByUsername(username)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	response := &UniqueUsernameResponse{
		Username: username,
		IsUnique: false,
	}

	if user == nil {
		response.IsUnique = true
	}

	render.Response(w, r, response, http.StatusOK)
}
