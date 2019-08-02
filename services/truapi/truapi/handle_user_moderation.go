package truapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/render"

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
	UserID     int64      `json:"user_id"`
	Moderation Moderation `json:"moderation"`
}

// HandleUserModeration handles the moderation of the users who have requested to signup
func (ta *TruAPI) HandleUserModeration(w http.ResponseWriter, r *http.Request) {
	var request ModerationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
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
		render.Error(w, r, fmt.Sprintf("moderation must either be '%s' or '%s' only", ModerationApproved, ModerationRejected), http.StatusBadRequest)
		return
	}
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}

func sendSignupEmail(ta *TruAPI, userID int64) error {
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
