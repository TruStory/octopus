package truapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AddCommentRequest represents the JSON request for adding a comment
type AddCommentRequest struct {
	ParentID   int64  `json:"parent_id,omitempty"`
	ClaimID    int64  `json:"claim_id,omitempty"`
	ArgumentID int64  `json:"argument_id,omitempty"`
	ElementID  int64  `json:"element_id,omitempty"`
	Body       string `json:"body"`
}

// HandleComment handles requests for comments
func (ta *TruAPI) HandleComment(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.handleCreateComment(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) handleCreateComment(r *http.Request) chttp.Response {
	request := &AddCommentRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	comment := &db.Comment{
		ParentID:   request.ParentID,
		ClaimID:    request.ClaimID,
		ArgumentID: request.ArgumentID,
		ElementID:  request.ElementID,
		Body:       request.Body,
		Creator:    user.Address,
	}
	err = ta.DBClient.AddComment(comment)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	respBytes, err := json.Marshal(comment)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	ta.sendCommentNotification(CommentNotificationRequest{
		ID:         comment.ID,
		ClaimID:    comment.ClaimID,
		ArgumentID: comment.ArgumentID,
		ElementID:  comment.ElementID,
		Creator:    comment.Creator,
		Timestamp:  time.Now(),
	})
	return chttp.SimpleResponse(200, respBytes)
}
