package truapi

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/TruStory/octopus/services/truapi/chttp"
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
func (ta *TruAPI) HandleUserForgotPassword(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.forgotPassword(r)
	case http.MethodPut:
		return ta.resetPassword(r)
	default:
		return chttp.SimpleErrorResponse(http.StatusMethodNotAllowed, errors.New("method not allowed"))
	}
}

func (ta *TruAPI) forgotPassword(r *http.Request) chttp.Response {
	var request ForgotPasswordRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	user, err := ta.DBClient.UserByEmailOrUsername(request.Identifier)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusNotFound, err)
	}
	if user == nil {
		return chttp.SimpleErrorResponse(http.StatusNotFound, errors.New("no such user"))
	}

	prt, err := ta.DBClient.IssueResetToken(user.ID)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	sendResetTokenToUser(ta, prt, user)

	return chttp.SimpleResponse(http.StatusOK, nil)
}

func (ta *TruAPI) resetPassword(r *http.Request) chttp.Response {
	var request PasswordResetRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	prt, err := ta.DBClient.UnusedResetTokenByUserAndToken(request.UserID, request.Token)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	if prt == nil {
		return chttp.SimpleErrorResponse(http.StatusNotFound, errors.New("no such token"))
	}

	err = ta.DBClient.UseResetToken(prt)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = ta.DBClient.ResetPassword(request.UserID, request.Password)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)
}

func sendResetTokenToUser(ta *TruAPI, prt *db.PasswordResetToken, user *db.User) {
	// TODO: send password reset email
}
