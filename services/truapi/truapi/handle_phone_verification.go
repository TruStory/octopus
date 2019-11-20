package truapi

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/twilio"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

const PhoneVerificationTokenLength = 6

type PhoneVerificationInitiateRequest struct {
	Phone string `json:"phone"`
}

type PhoneVerificationVerifyRequest struct {
	Token string `json:"token"`
}

// HandlePhoneVerification verifies the phone of user
func (ta *TruAPI) HandlePhoneVerification(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := ta.DBClient.UserByID(authenticatedUser.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}
	if user == nil {
		render.Error(w, r, "user not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPost:
		ta.initiatePhoneVerification(w, r, user)
	case http.MethodPut:
		ta.verifyPhone(w, r, user)
	default:
		render.Error(w, r, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ta *TruAPI) initiatePhoneVerification(w http.ResponseWriter, r *http.Request, user *db.User) {
	if user.PhoneVerifiedAt != nil {
		render.Error(w, r, "already verified", http.StatusBadRequest)
		return
	}

	var request PhoneVerificationInitiateRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user.VerifiedPhoneHash = fmt.Sprintf("%x", (md5.Sum([]byte(request.Phone)))) // md5 hash of the phone
	user.PhoneVerificationToken = generateRandomToken(PhoneVerificationTokenLength)

	// sending the message
	client := twilio.NewClient(
		ta.APIContext.Config.Twilio.AccountSID,
		ta.APIContext.Config.Twilio.AuthToken,
		ta.APIContext.Config.Twilio.From,
	)
	msg, err := twilio.NewMessage("verification", user.PhoneVerificationToken)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = client.Send(request.Phone, msg)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ta.DBClient.UpdateModel(user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Response(w, r, true, http.StatusOK)
}

func (ta *TruAPI) verifyPhone(w http.ResponseWriter, r *http.Request, user *db.User) {
	if user.PhoneVerifiedAt != nil {
		render.Error(w, r, "already verified", http.StatusBadRequest)
		return
	}

	var request PhoneVerificationVerifyRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	if user.PhoneVerificationToken != request.Token {
		render.Error(w, r, "invalid token", http.StatusBadRequest)
		return
	}

	now := time.Now()
	user.PhoneVerifiedAt = &now
	err = ta.DBClient.UpdateModel(user)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Response(w, r, true, http.StatusOK)
}

func generateRandomToken(length int) string {
	token := ""
	for i := 0; i < length; i++ {
		rand.Seed(time.Now().UnixNano())
		token += strconv.Itoa(rand.Intn(9))
	}

	return token
}
