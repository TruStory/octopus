package db

import (
	"fmt"
	"time"
)

// NotificationType represents a type of notification defiend by the system.
type NotificationType int

// NotificationsCountResponse is the interface to respond graphQL query
type NotificationsCountResponse struct {
	Count int64 `json:"count"`
}

// CoinDisplayName is the name of the coin presented to end users.
const CoinDisplayName = "TRU"

// Types of notifications.
const (
	NotificationStoryAction    NotificationType = iota // deprecated
	NotificationArgumentAction                         // deprecated
	NotificationCommentAction
	NotificationMentionAction
	NotificationNewArgument
	NotificationAgreeReceived
	NotificationNotHelpful
	NotificationEarnedStake
	NotificationSlashed
	NotificationJailed
	NotificationUnjailed
	NotificationArgumentCommentAction
)

var NotificationTypeName = []string{
	NotificationStoryAction:           "Story Update",
	NotificationArgumentAction:        "Argument Update",
	NotificationCommentAction:         "Reply Added",
	NotificationMentionAction:         "Mentioned",
	NotificationNewArgument:           "New Argument",
	NotificationAgreeReceived:         "Agree received on Argument",
	NotificationNotHelpful:            "Not Helpful received on Argument",
	NotificationEarnedStake:           fmt.Sprintf("Earned %s", CoinDisplayName),
	NotificationSlashed:               "Slashed",
	NotificationJailed:                "Jailed",
	NotificationUnjailed:              "Unjailed",
	NotificationArgumentCommentAction: "Reply Added",
}

func (t NotificationType) String() string {
	if int(t) >= len(NotificationTypeName) {
		return ""
	}
	return NotificationTypeName[t]
}

// MentionType represents  the types on how an user can be mentioned.
type MentionType int64

// Types of mentions.
const (
	MentionArgument MentionType = iota
	MentionComment
	MentionArgumentComment
)

var MentionTypeName = []string{
	MentionArgument: "in an Argument",
	MentionComment:  "in a Reply",
}

func (t MentionType) String() string {
	if int(t) >= len(MentionTypeName) {
		return ""
	}
	return MentionTypeName[t]
}

// NotificationMeta  contains extra payload information.
type NotificationMeta struct {
	ClaimID     *int64       `json:"claimId,omitempty" graphql:"claimId"`
	ArgumentID  *int64       `json:"argumentId,omitempty" graphql:"argumentId"`
	ElementID   *int64       `json:"elementId,omitempty" graphql:"elementId"`
	StoryID     *int64       `json:"storyId,omitempty" graphql:"storyId"`
	CommentID   *int64       `json:"commentId,omitempty" graphql:"commentId"`
	MentionType *MentionType `json:"mentionType,omitempty" graphql:"mentionType"`
}

// NotificationEvent represents a notification sent to an user.
type NotificationEvent struct {
	Timestamps
	ID              int64            `json:"id"`
	TypeID          int64            `json:"type_id"`
	Address         string           `json:"address"`
	UserProfileID   int64            `json:"user_profile_id"`
	UserProfile     *User            `json:"user_profile"`
	Message         string           `json:"message"`
	Timestamp       time.Time        `json:"timestamp"`
	SenderProfileID int64            `json:"sender_profile_id" `
	SenderProfile   *User            `json:"sender_profile"`
	Type            NotificationType `json:"type" sql:",notnull"`
	Meta            NotificationMeta `json:"meta"`
	Read            bool             `json:"read"`
	Seen            bool             `json:"seen"`
}

// NotificationEventsByAddress retrieves all notifications sent to an user.
// TODO (issue #435): add pagination
func (c *Client) NotificationEventsByAddress(addr string) ([]NotificationEvent, error) {
	evts := make([]NotificationEvent, 0)

	err := c.Model(&evts).
		Column("notification_event.*", "UserProfile", "SenderProfile").
		Where("notification_event.address = ?", addr).Order("timestamp DESC").Select()
	if err != nil {
		return nil, err
	}
	return evts, nil
}

// UnreadNotificationEventsCountByAddress retrieves the number of unread notifications sent to an user.
func (c *Client) UnreadNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error) {
	notificationEvent := new(NotificationEvent)

	count, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("read is NULL or read is FALSE").Count()
	if err != nil {
		return &NotificationsCountResponse{
			Count: 0,
		}, err
	}

	return &NotificationsCountResponse{
		Count: int64(count),
	}, nil
}

// UnseenNotificationEventsCountByAddress retrieves the number of unseen notifications sent to an user.
func (c *Client) UnseenNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error) {
	notificationEvent := new(NotificationEvent)

	count, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("seen is NULL or seen is FALSE").Count()
	if err != nil {
		return &NotificationsCountResponse{
			Count: 0,
		}, err
	}

	return &NotificationsCountResponse{
		Count: int64(count),
	}, nil
}

// MarkAllNotificationEventsAsReadByAddress marks all notifications read for a given user.
func (c *Client) MarkAllNotificationEventsAsReadByAddress(addr string) error {
	notificationEvent := new(NotificationEvent)

	_, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("read is NULL or read is FALSE").
		Set("read = ?", true).
		Update()
	if err != nil {
		return err
	}

	return nil
}

// MarkAllNotificationEventsAsSeenByAddress marks all notifications seen for a given user.
func (c *Client) MarkAllNotificationEventsAsSeenByAddress(addr string) error {
	notificationEvent := new(NotificationEvent)

	_, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("seen is NULL or seen is FALSE").
		Set("seen = ?", true).
		Update()
	if err != nil {
		return err
	}

	return nil
}

// MarkThreadNotificationsAsRead mark previous notification replies of the same thread as read.
func (c *Client) MarkThreadNotificationsAsRead(addr string, claimID int64) error {
	notificationEvent := new(NotificationEvent)
	_, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("notification_event.type = ?", NotificationCommentAction).
		Where("(notification_event.meta->>'claimId')::integer = ?", claimID).
		Set("read = ?", true).
		Update()
	if err != nil {
		return err
	}
	return nil
}

// MarkArgumentNotificationAsRead marks as read argument notification.
func (c *Client) MarkArgumentNotificationAsRead(addr string, claimID int64, argumentID int64) error {
	notificationEvent := new(NotificationEvent)
	_, err := c.Model(notificationEvent).
		Where("notification_event.address = ?", addr).
		Where("notification_event.type = ?", NotificationNewArgument).
		Where("(notification_event.meta->>'claimId')::integer = ?", claimID).
		Where("(notification_event.meta->>'argumentId')::integer = ?", argumentID).
		Set("read = ?", true).
		Update()
	if err != nil {
		return err
	}
	return nil
}
