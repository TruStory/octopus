package truapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman/messages"
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

// TruErrors for handle user
var (
	ErrNoSuchUser  = render.TruError{Code: 400, Message: "No such user."}
	ErrNoSuchToken = render.TruError{Code: 401, Message: "No such token."}
	ErrTwitterUser = render.TruError{Code: 402, Message: "Please use Twitter to reset this password."}
)

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
		render.LoginError(w, r, ErrNoSuchUser, http.StatusNotFound)
		return
	}


	isTwitterUser := ta.DBClient.IsTwitterUser((*user).ID)
	if isTwitterUser {
		render.LoginError(w, r, ErrTwitterUser, http.StatusBadRequest)
		return
	}

	user, err = ta.DBClient.VerifiedUserByID((*user).ID)
	if err != nil || user == nil {
		render.LoginError(w, r, ErrEmailNotVerified, http.StatusBadRequest)
		return
	}

	prt, err := ta.DBClient.IssueResetToken(user.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = sendResetTokenToUser(ta, prt, user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
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
		render.LoginError(w, r, ErrNoSuchToken, http.StatusNotFound)
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
}

func sendResetTokenToUser(ta *TruAPI, prt *db.PasswordResetToken, user *db.User) error {
	message, err := messages.MakePasswordResetMessage(ta.Postman, ta.APIContext.Config, *user, *prt)
	if err != nil {
		return errors.New("error sending password reset email")
	}

	err = ta.Postman.Deliver(*message)
	if err != nil {
		return errors.New("error sending password reset email")
	}
	return nil
}
