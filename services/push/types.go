package main

import (
	"time"

	db "github.com/TruStory/octopus/services/api/db"
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
	ID              int64     `json:"id"`
	ArgumentCreator string    `json:"argument_creator"`
	ArgumentID      int64     `json:"argumentId"`
	StoryID         int64     `json:"storyId"`
	Creator         string    `json:"creator"`
	Timestamp       time.Time `json:"timestamp"`
}

// GraphQL responses

// Staker represents the user that backed/challenge.
type Staker struct {
	Creator struct {
		Address string `json:"address"`
	}
}

// StoryParticipants contains challenges and backings.
type StoryParticipants struct {
	Creator struct {
		Address string `json:"address"`
	} `json:"creator"`
	Backings   []Staker `json:"backings"`
	Challenges []Staker `json:"challenges"`
}

// StoryParticipantsResponse is the response from the graphql endpoint.
type StoryParticipantsResponse struct {
	Story StoryParticipants `json:"story"`
}

// StoryParticipantsQuery is the GraphQL query to get all address staking in a story.
const StoryParticipantsQuery = `
  query Story($storyId: ID!) {
	story(iD: $storyId) {
	  creator {
		address
	  }
	  backings {
		creator {
		  address
		}
	  }
	  challenges {
		creator {
		  address
		}
	  }
	}
  }
  
`

// Argument represents the argument.
type Argument struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	StoryID int64  `json:"storyId"`
}

// BackingResponse is the response from the graphql endpoint.
type BackingResponse struct {
	BackingArgument struct {
		Argument Argument `json:"argument"`
	} `json:"backing"`
}

// ChallengeResponse is the response from the graphql endpoint.
type ChallengeResponse struct {
	ChallengeArgument struct {
		Argument Argument `json:"argument"`
	} `json:"challenge"`
}

// ArgumentByStakeIDQuery is the GraphQL query to get the argument tied to a stake (backing/challenge)
const ArgumentByStakeIDQuery = ` 
	query ArgumentByStakeID($id: ID!) {
		%s(iD: $id) {
			argument(raw: true) {
				id
				body
				storyId
			}
		}
	}
`
