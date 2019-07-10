package main

import (
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/appleboy/gorush/gorush"
)

// Notification represents a parsed event comming from the chain.
type Notification struct {
	From   *string
	To     string
	Msg    string
	TypeID int64
	Type   db.NotificationType
	Meta   db.NotificationMeta
	Action string
	Trim   bool
}

// NotificationData represents the data relevant to the app.
type NotificationData struct {
	ID int64 `json:"id"`
	// StoryID
	TypeID    int64               `json:"typeId"`
	Timestamp time.Time           `json:"timestamp"`
	Read      bool                `json:"read"`
	Type      db.NotificationType `json:"type"`
	// UserID is the sender
	UserID *string             `json:"userId,omitempty"`
	Image  *string             `json:"image,omitempty"`
	Meta   db.NotificationMeta `json:"meta"`
}

// ToGorushData translate to gorush data format.
func (d NotificationData) ToGorushData() gorush.D {
	data := make(map[string]interface{})
	// embbed everything inside a trustory key
	data["trustory"] = d
	return data
}

// PushNotification represents the notification data sent to services.
type PushNotification struct {
	Title            string
	Subtitle         string
	Body             string
	Platform         string
	NotificationData NotificationData
}

// GorushResponse represents a json payload response.
type GorushResponse struct {
	Success string                `json:"success"`
	Counts  int                   `json:"counts"`
	Logs    []gorush.LogPushEntry `json:"logs"`
}

// CommentNotificationRequest is the payload sent to pushd for sending notifications.
type CommentNotificationRequest struct {
	// ID is the comment id.
	ID           int64     `json:"id"`
	ClaimCreator string    `json:"claim_creator"`
	ClaimID      int64     `json:"claimId"`
	ArgumentID   int64     `json:"argumentId"`
	Creator      string    `json:"creator"`
	Timestamp    time.Time `json:"timestamp"`
}

// GraphQL responses

const ClaimArgumentByIDQuery = `
query ClaimArgumentQuery($argumentId: ID!) {
  claimArgument(id: $argumentId) {
    id
    claimId
    claim {
      body
      id
      creator {
        address
      }
      participants {
        address
      }
    }
  }
}
`

const argumentSummaryByIDQuery = `
query ClaimArgumentQuery($argumentId: ID!) {
  claimArgument(id: $argumentId) {
    id
    claimId
    summary
	creator{
      address
    }
  }
}
`

// Creator represents the user that backed/challenge.
type Creator struct {
	Address string `json:"address"`
}

// ClaimArgumentResponse is the response from the graphql endpoint.
type ClaimArgumentResponse struct {
	ClaimArgument struct {
		ID      int64 `json:"id"`
		ClaimID int64 `json:"claimId"`
		Claim   struct {
			Body         string    `json:"body"`
			Creator      Creator   `json:"creator"`
			Participants []Creator `json:"participants"`
		} `json:"claim"`
	} `json:"claimArgument"`
}

// ArgumentSummaryResponse is the response from the graphql endpoint.
type ArgumentSummaryResponse struct {
	ClaimArgument struct {
		ID      int64   `json:"id"`
		ClaimID int64   `json:"claimId"`
		Creator Creator `json:"creator"`
		Summary string  `json:"summary"`
	} `json:"claimArgument"`
}
