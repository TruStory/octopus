package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// AuthenticationRequest represents the http request to authenticate a user's account
type AuthenticationRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// TruErrors for user auth
var (
	ErrServerError        = render.TruError{Code: 300, Message: "Server Error. Please try again later."}
	ErrUnverifiedEmail    = render.TruError{Code: 301, Message: "Please verify your email."}
	ErrInvalidCredentials = render.TruError{Code: 302, Message: "Invalid login credentials."}
)

// HandleUserAuthentication handles the moderation of the users who have requested to signup
func (ta *TruAPI) HandleUserAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request AuthenticationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.LoginError(w, r, ErrServerError, http.StatusInternalServerError)
		return
	}

	user, err := ta.DBClient.GetAuthenticatedUser(request.Identifier, request.Password)
	if err != nil {
		render.LoginError(w, r, ErrInvalidCredentials, http.StatusBadRequest)
		return
	}

	if (*user).VerifiedAt.IsZero() {
		render.LoginError(w, r, ErrUnverifiedEmail, http.StatusBadRequest)
		return
	}

	cookie, err := cookies.GetLoginCookie(ta.APIContext, user)
	if err != nil {
		render.LoginError(w, r, ErrServerError, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie)
	render.Response(w, r, user, http.StatusOK)
}
