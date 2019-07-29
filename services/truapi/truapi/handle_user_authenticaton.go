package truapi

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AuthenticationRequest represents the http request to authenticate a user's account
type AuthenticationRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
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
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		err = json.Unmarshal(reqBody, &request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		user, err := ta.DBClient.GetAuthenticatedUser(request.Email, request.Username, request.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cookie, err := cookies.GetEmailLoginCookie(ta.APIContext, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusNoContent)
	}

	return http.HandlerFunc(fn)
}
