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
	ClaimID    *int64 `json:"claimId,omitempty"`
	ArgumentID *int64 `json:"argumentId,omitempty"`
}

// TrackEvent represents an event that is tracked
type TrackEvent struct {
	Event      string          `json:"event"`
	Properties EventProperties `json:"properties"`
}

// TrackEventClaimOpened event tracks opened claims
const (
	TrackEventClaimOpened    = "claim_opened"
	TrackEventArgumentOpened = "argument_opened"
)

// HandleTrackEvent records an event in the database
func (ta *TruAPI) HandleTrackEvent(w http.ResponseWriter, r *http.Request) {
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
	var dbEvent db.TrackEvent

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
		dbEvent = db.TrackEvent{
			Event: TrackEventClaimOpened,
			Meta: db.TrackEventMeta{
				ClaimID:     evt.Properties.ClaimID,
				CommunityID: &claim.CommunityID,
			},
		}
	case TrackEventArgumentOpened:
		if evt.Properties.ClaimID == nil {
			fmt.Println("empty claim id")
			w.WriteHeader(http.StatusOK)
			return
		}
		if evt.Properties.ArgumentID == nil {
			fmt.Println("empty argument id")
			w.WriteHeader(http.StatusOK)
			return
		}
		claim := ta.claimResolver(r.Context(), queryByClaimID{ID: uint64(*evt.Properties.ClaimID)})
		if claim.ID == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}
		dbEvent = db.TrackEvent{
			Event: TrackEventArgumentOpened,
			Meta: db.TrackEventMeta{
				ClaimID:     evt.Properties.ClaimID,
				ArgumentID:  evt.Properties.ArgumentID,
				CommunityID: &claim.CommunityID,
			},
		}
	}

	if dbEvent.Event == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if ok && user != nil {
		dbEvent.Address = user.Address
		dbEvent.TwitterProfileID = user.TwitterProfileID
	}

	if user == nil {
		sess, err := cookies.GetAnonymousSession(ta.APIContext, r)
		if err != nil {
			fmt.Println("unable to get session id cookie")
			w.WriteHeader(http.StatusOK)
			return
		}
		dbEvent.IsAnonymous = true
		dbEvent.SessionID = sess.SessionID
	}
	err = ta.DBClient.Add(&dbEvent)
	if err != nil {
		fmt.Println("error adding track event", err)
	}

	w.WriteHeader(http.StatusOK)
}
