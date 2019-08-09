package truapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/postman/messages"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// ResendEmailVerificationRequest represents the request to resend the email verification
type ResendEmailVerificationRequest struct {
	Identifier string `json:"identifier"`
}

// TruErrors for resend email verification
var (
	ErrUserNotFound        = render.TruError{Code: 200, Message: "No such user found."}
	ErrUserAlreadyVerified = render.TruError{Code: 201, Message: "User is already verified."}
)

// HandleResendEmailVerification resends the email verification email
func (ta *TruAPI) HandleResendEmailVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		render.Error(w, r, "method not allowed", http.StatusMethodNotAllowed)
	}

	var request ResendEmailVerificationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := ta.DBClient.UserByEmailOrUsername(request.Identifier)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if user == nil {
		render.LoginError(w, r, ErrUserNotFound, http.StatusBadRequest)
		return
	}
	if !user.VerifiedAt.IsZero() {
		render.LoginError(w, r, ErrUserAlreadyVerified, http.StatusBadRequest)
		return
	}

	message, err := messages.MakeEmailConfirmationMessage(ta.Postman, ta.APIContext.Config, *user)
	if err != nil {
		fmt.Println("could not remake verification email: ", user, err)
		render.Error(w, r, "cannot send email confirmation right now", http.StatusInternalServerError)
		return
	}

	err = ta.Postman.Deliver(*message)
	if err != nil {
		fmt.Println("could not resend verification email: ", user, err)
		render.Error(w, r, "cannot send email confirmation right now", http.StatusInternalServerError)
		return
	}
}
