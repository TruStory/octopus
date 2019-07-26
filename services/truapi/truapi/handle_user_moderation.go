package truapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman/messages"
)

// Moderation represents the moderation done for a user
type Moderation string

const (
	// ModerationApproved is to represent the approval
	ModerationApproved Moderation = "approved"
	// ModerationRejected is to represent the rejection
	ModerationRejected Moderation = "rejected"
)

// ModerationRequest represents the http request to moderate a user's request to signup
type ModerationRequest struct {
	UserID     uint64     `json:"user_id"`
	Moderation Moderation `json:"moderation"`
}

// HandleUserModeration handles the moderation of the users who have requested to signup
func (ta *TruAPI) HandleUserModeration(r *http.Request) chttp.Response {
	var request ModerationRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	// TODO: make sure only admins can take this action
	if request.Moderation == ModerationApproved {
		err = ta.DBClient.ApproveUserByID(request.UserID)
		// if approved, send them a signup email
		if err == nil {
			err = sendSignupEmail(ta, request.UserID)
		}
	} else if request.Moderation == ModerationRejected {
		err = ta.DBClient.RejectUserByID(request.UserID)
	} else {
		return chttp.SimpleErrorResponse(http.StatusBadRequest, fmt.Errorf("moderation must either be '%s' or '%s' only", ModerationApproved, ModerationRejected))
	}
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)
}

func sendSignupEmail(ta *TruAPI, userID uint64) error {
	user := db.User{ID: userID}
	err := ta.DBClient.Find(&user)
	if err != nil {
		return err
	}

	message, err := messages.MakeSignupMessage(ta.Postman, ta.APIContext.Config, user)
	if err != nil {
		return err
	}

	err = ta.Postman.Deliver(*message)
	if err != nil {
		return err
	}

	return nil
}
