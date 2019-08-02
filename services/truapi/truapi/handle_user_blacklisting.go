package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UserBlacklistRequest represents the http request to blacklist a user
type UserBlacklistRequest struct {
	UserID int64 `json:"user_id"`
}

// HandleUserBlacklisting blacklists the bad actors on the platform
func (ta *TruAPI) HandleUserBlacklisting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		render.Error(w, r, "method not method", http.StatusMethodNotAllowed)
		return
	}

	var request UserBlacklistRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = ta.DBClient.BlacklistUser(request.UserID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}
