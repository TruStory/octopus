package db

import (
	"context"
	"time"

	"github.com/go-pg/pg/orm"
)

// Datastore defines all operations on the DB
// This interface can be mocked out for tests, etc.
type Datastore interface {
	Mutations
	Queries
}

// Mutations write to the database
type Mutations interface {
	GenericMutations
	UpsertTwitterProfile(profile *TwitterProfile) error
	UpsertDeviceToken(token *DeviceToken) error
	RemoveDeviceToken(address, token, platform string) error
	UpsertFlaggedStory(flaggedStory *FlaggedStory) error
	MarkAllNotificationEventsAsReadByAddress(addr string) error
	MarkAllNotificationEventsAsSeenByAddress(addr string) error
	AddComment(comment *Comment) error
	AddInvite(invite *Invite) error
	ReactOnReactionable(addr string, reaction ReactionType, reactionable Reactionable) error
	UnreactByAddressAndID(addr string, id int64) error
	AddClaimOfTheDayID(claimOfTheDayID *ClaimOfTheDayID) error
	DeleteClaimOfTheDayID(communityID string) error
	AddClaimImage(claimImage *ClaimImage) error
	AddUser(user *User) error
	ApproveUserByID(id uint64) error
	RejectUserByID(id uint64) error
	SignupUser(id uint64, token string, username string, password string) error
	UpdatePassword(id uint64, password *UserPassword) error
	UpdateProfile(id uint64, profile *UserProfile) error
}

// Queries read from the database
type Queries interface {
	GenericQueries
	TwitterProfileByID(id int64) (TwitterProfile, error)
	TwitterProfileByAddress(addr string) (*TwitterProfile, error)
	TwitterProfileByUsername(username string) (*TwitterProfile, error)
	UsernamesByPrefix(prefix string) ([]string, error)
	KeyPairByTwitterProfileID(id int64) (KeyPair, error)
	DeviceTokensByAddress(addr string) ([]DeviceToken, error)
	NotificationEventsByAddress(addr string) ([]NotificationEvent, error)
	UnreadNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error)
	UnseenNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error)
	FlaggedStoriesIDs(flagAdmin string, flagLimit int) ([]int64, error)
	CommentsByArgumentID(argumentID int64) ([]Comment, error)
	CommentsByClaimID(claimID uint64) ([]Comment, error)
	Invites() ([]Invite, error)
	InvitesByAddress(addr string) ([]Invite, error)
	ReactionsByReactionable(reactionable Reactionable) ([]Reaction, error)
	ReactionsByAddress(addr string) ([]Reaction, error)
	ReactionsCountByReactionable(reactionable Reactionable) ([]ReactionsCount, error)
	TranslateToCosmosMentions(body string) (string, error)
	TranslateToUsersMentions(body string) (string, error)
	InitialStakeBalanceByAddress(address string) (*InitialStakeBalance, error)
	OpenedClaimsSummary(date time.Time) ([]UserOpenedClaimsSummary, error)
	ClaimOfTheDayIDByCommunityID(communityID string) (int64, error)
	ClaimImageURL(claimID uint64) (string, error)
	SignedupUserByID(id uint64) (*User, error)
	UnsignedupUserByIDAndToken(id uint64, token string) (*User, error)
	GetAuthenticatedUser(email, username, password string) (*User, error)
	UserByEmail(email string) (*User, error)
	UserByUsername(username string) (*User, error)
}

// Timestamps carries the default timestamp fields for any derived model
type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// BeforeInsert is the hook that fills in the created_at and updated_at fields
func (m *Timestamps) BeforeInsert(ctx context.Context, db orm.DB) error {
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate is the hook that updates the updated_at field
func (m *Timestamps) BeforeUpdate(ctx context.Context, db orm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}
