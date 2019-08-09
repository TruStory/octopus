package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AuthenticationRequest represents the http request to authenticate a user's account
type AuthenticationRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// HandleUserAuthentication handles the moderation of the users who have requested to signup
func HandleUserAuthentication(ta *TruAPI) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// only support POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request AuthenticationRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		user, err := ta.DBClient.GetAuthenticatedUser(request.Identifier, request.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if (*user).VerifiedAt.IsZero() {
			http.Error(w, "please verify your email", http.StatusBadRequest)
			return
		}

		cookie, err := cookies.GetLoginCookie(ta.APIContext, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusNoContent)
	}

	return http.HandlerFunc(fn)
}
