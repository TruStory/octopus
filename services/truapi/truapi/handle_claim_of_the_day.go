package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
)

// AddClaimOfTheDayIDRequest represents the JSON request for adding a claim of the day id
type AddClaimOfTheDayIDRequest struct {
	CommunityID string `json:"community_id"`
	ClaimID     int64  `json:"claim_id"`
}

// DeleteClaimOfTheDayIDRequest represents the JSON request for deleting a claim of the day id
type DeleteClaimOfTheDayIDRequest struct {
	CommunityID string `json:"community_id"`
}

// HandleClaimOfTheDayID handles requests for claim of the day
func (ta *TruAPI) HandleClaimOfTheDayID(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.addClaimOfTheDayID(r)
	case http.MethodDelete:
		return ta.deleteClaimOfTheDayID(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) addClaimOfTheDayID(r *http.Request) chttp.Response {
	request := &AddClaimOfTheDayIDRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user := r.Context().Value(userContextKey)
	if user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	claimOfTheDayID := &db.ClaimOfTheDayID{
		CommunityID: request.CommunityID,
		ClaimID:     request.ClaimID,
	}
	err = ta.DBClient.AddClaimOfTheDayID(claimOfTheDayID)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	if request.CommunityID == "all" {
		ta.sendBroadcastNotification(BroadcastNotificationRequest{
			Type: db.NotificationFeaturedDebate,
		})
	}
	return chttp.SimpleResponse(200, nil)
}

func (ta *TruAPI) deleteClaimOfTheDayID(r *http.Request) chttp.Response {
	request := &DeleteClaimOfTheDayIDRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user := r.Context().Value(userContextKey)
	if user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	err = ta.DBClient.DeleteClaimOfTheDayID(request.CommunityID)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	return chttp.SimpleResponse(200, nil)
}
