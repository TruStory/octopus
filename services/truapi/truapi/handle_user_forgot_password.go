package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// ForgotPasswordRequest represents the http request when a user forgets a password
type ForgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

// PasswordResetRequest represents the http request for a user to reset their password
type PasswordResetRequest struct {
	UserID   int64  `json:"user_id"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

// HandleUserForgotPassword handles the resetting of user's password if they forget
func (ta *TruAPI) HandleUserForgotPassword(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ta.forgotPassword(w, r)
	case http.MethodPut:
		ta.resetPassword(w, r)
	default:
		render.Error(w, r, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ta *TruAPI) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var request ForgotPasswordRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := ta.DBClient.UserByEmailOrUsername(request.Identifier)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusNotFound)
		return
	}
	if user == nil {
		render.Error(w, r, "no such user", http.StatusNotFound)
		return
	}

	prt, err := ta.DBClient.IssueResetToken(user.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	sendResetTokenToUser(ta, prt, user)

	render.Response(w, r, true, http.StatusOK)
	return
}

func (ta *TruAPI) resetPassword(w http.ResponseWriter, r *http.Request) {
	var request PasswordResetRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = validatePassword(request.Password)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	prt, err := ta.DBClient.UnusedResetTokenByUserAndToken(request.UserID, request.Token)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if prt == nil {
		render.Error(w, r, "no such token", http.StatusNotFound)
		return
	}

	err = ta.DBClient.UseResetToken(prt)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	err = ta.DBClient.ResetPassword(request.UserID, request.Password)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
	return
}

func sendResetTokenToUser(ta *TruAPI, prt *db.PasswordResetToken, user *db.User) {
	// TODO: send password reset email
}
