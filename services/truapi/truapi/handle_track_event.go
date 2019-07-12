package truapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"encoding/base64"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// EventProperties holds tracking event information
type EventProperties struct {
	ClaimID *int64 `json:"claimId,omitempty"`
}

// TrackEvent represents an event that is tracked
type TrackEvent struct {
	Event      string          `json:"event"`
	Properties EventProperties `json:"properties"`
}

// TrackEventClaimOpened event tracks opened claims
const (
	TrackEventClaimOpened = "claim_opened"
)

// HandleTrackEvent records an event in the database
func (ta *TruAPI) HandleTrackEvent(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	// ignore not logged in users for now
	if !ok || user == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	b, err := base64.StdEncoding.DecodeString(r.FormValue("data"))
	if err != nil {
		fmt.Println("error decoding event", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	evt := &TrackEvent{}
	err = json.Unmarshal(b, evt)
	if err != nil {
		fmt.Println("error decoding event", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch evt.Event {
	case TrackEventClaimOpened:
		if evt.Properties.ClaimID == nil {
			fmt.Println("empty claim id")
			w.WriteHeader(http.StatusOK)
			return
		}
		claim := ta.claimResolver(r.Context(), queryByClaimID{ID: uint64(*evt.Properties.ClaimID)})
		if claim.ID == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}
		claimID := int64(claim.ID)
		dbEvent := db.TrackEvent{
			Address:          user.Address,
			TwitterProfileID: user.TwitterProfileID,
			Event:            TrackEventClaimOpened,
			Meta: db.TrackEventMeta{
				ClaimID:     &claimID,
				CommunityID: &claim.CommunityID,
			},
		}
		err := ta.DBClient.Add(&dbEvent)
		if err != nil {
			fmt.Println("error adding track event", err)
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
}
