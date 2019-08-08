package truapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/TruStory/octopus/services/truapi/postman/messages"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AddInviteRequest represents the JSON request for adding an invite
type AddInviteRequest struct {
	Email string `json:"email"`
}

// HandleInvite handles requests for invites
func (ta *TruAPI) HandleInvite(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.handleCreateInvite(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) handleCreateInvite(r *http.Request) chttp.Response {
	request := &AddInviteRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}
	// check if valid email address
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if !re.MatchString(request.Email) {
		return chttp.SimpleErrorResponse(422, errors.New("Invalid email address"))
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	invite := &db.Invite{
		Creator:     user.Address,
		FriendEmail: request.Email,
	}
	err = ta.DBClient.AddInvite(invite)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	if invite.ID == 0 {
		return chttp.SimpleErrorResponse(422, errors.New("This user has already been invited"))
	}
	// send invitation via email
	err = sendInvitationToFriend(ta, invite.FriendEmail, user.ID)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	respBytes, err := json.Marshal(invite)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	return chttp.SimpleResponse(200, respBytes)
}

func sendInvitationToFriend(ta *TruAPI, friend string, referrerID int64) error {
	referrer := db.User{ID: referrerID}
	err := ta.DBClient.Find(&referrer)
	if err != nil {
		return errors.New("error sending invitation to the friend")
	}
	message, err := messages.MakeInvitationMessage(ta.Postman, ta.APIContext.Config, friend, referrer)
	if err != nil {
		return errors.New("error sending invitation to the friend")
	}

	err = ta.Postman.Deliver(*message)
	if err != nil {
		return errors.New("error sending invitation to the friend")
	}
	return nil
}
