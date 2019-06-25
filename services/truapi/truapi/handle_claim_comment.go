package truapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AddClaimCommentRequest represents the JSON request for adding a claim comment
type AddClaimCommentRequest struct {
	ParentID int64  `json:"parent_id,omitonempty"`
	ClaimID  int64  `json:"claim_id"`
	Body     string `json:"body"`
}

// HandleClaimComment handles requests for claim comments
func (ta *TruAPI) HandleClaimComment(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.handleCreateClaimComment(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) handleCreateClaimComment(r *http.Request) chttp.Response {
	request := &AddClaimCommentRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	comment := &db.ClaimComment{
		ParentID: request.ParentID,
		ClaimID:  request.ClaimID,
		Body:     request.Body,
		Creator:  user.Address,
	}
	err = ta.DBClient.AddClaimComment(comment)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	respBytes, err := json.Marshal(comment)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	ta.sendClaimCommentNotification(ClaimCommentNotificationRequest{
		ID:        comment.ID,
		ClaimID:   comment.ClaimID,
		Creator:   comment.Creator,
		Timestamp: time.Now(),
	})
	return chttp.SimpleResponse(200, respBytes)
}
