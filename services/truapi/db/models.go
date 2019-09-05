package db

import (
	"context"
	"time"

	"github.com/go-pg/pg"
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
	UpsertDeviceToken(token *DeviceToken) error
	RemoveDeviceToken(address, token, platform string) error
	UpsertFlaggedStory(flaggedStory *FlaggedStory) error
	MarkAllNotificationEventsAsReadByAddress(addr string) error
	MarkAllNotificationEventsAsSeenByAddress(addr string) error
	MarkThreadNotificationsAsRead(addr string, claimID int64) error
	MarkArgumentNotificationAsRead(addr string, claimID int64, argumentID int64) error
	AddComment(comment *Comment) error
	AddQuestion(question *Question) error
	DeleteQuestion(ID int64) error
	AddInvite(invite *Invite) error
	ReactOnReactionable(addr string, reaction ReactionType, reactionable Reactionable) error
	UnreactByAddressAndID(addr string, id int64) error
	AddClaimOfTheDayID(claimOfTheDayID *ClaimOfTheDayID) error
	DeleteClaimOfTheDayID(communityID string) error
	AddClaimImage(claimImage *ClaimImage) error
	AddUser(user *User) error
	ApproveUserByID(id int64) error
	RejectUserByID(id int64) error
	RegisterUser(user *User, referrerCode string) error
	BlacklistUser(id int64) error
	UnblacklistUser(id int64) error
	VerifyUser(id int64, token string) error
	TouchLastAuthenticatedAt(id int64) error
	AddAddressToUser(id int64, address string) error
	UpdatePassword(id int64, password *UserPassword) error
	ResetPassword(id int64, password string) error
	UpdateProfile(id int64, profile *UserProfile) error
	SetUserCredentials(id int64, credentials *UserCredentials) error
	SetUserMeta(id int64, userMeta *UserMeta) error
	IssueResetToken(userID int64) (*PasswordResetToken, error)
	UseResetToken(prt *PasswordResetToken) error
	UpsertConnectedAccount(connectedAccount *ConnectedAccount) error
	AddUserViaConnectedAccount(connectedAccount *ConnectedAccount) (*User, error)
	FollowCommunities(address string, communities []string) error
	FollowedCommunities(address string) ([]FollowedCommunity, error)
	UnfollowCommunity(address, communityID string) error
	FollowsCommunity(address, communityID string) (bool, error)
	AddImageURLToHighlight(id int64, url string) error
}

// Queries read from the database
type Queries interface {
	GenericQueries
	UsernamesByPrefix(prefix string) ([]string, error)
	KeyPairByUserID(userID int64) (*KeyPair, error)
	DeviceTokensByAddress(addr string) ([]DeviceToken, error)
	NotificationEventsByAddress(addr string) ([]NotificationEvent, error)
	UnreadNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error)
	UnseenNotificationEventsCountByAddress(addr string) (*NotificationsCountResponse, error)
	FlaggedStoriesIDs(flagAdmin string, flagLimit int) ([]int64, error)
	ArgumentLevelComments(argumentID uint64, elementID uint64) ([]Comment, error)
	CommentsByClaimID(claimID uint64) ([]Comment, error)
	ClaimLevelComments(claimID uint64) ([]Comment, error)
	CommentByID(id int64) (*Comment, error)
	QuestionsByClaimID(claimID uint64) ([]Question, error)
	QuestionByID(ID int64) (*Question, error)
	Invites() ([]Invite, error)
	InvitesByAddress(addr string) ([]Invite, error)
	InvitesByFriendEmail(email string) (*Invite, error)
	ReactionsByReactionable(reactionable Reactionable) ([]Reaction, error)
	ReactionsByAddress(addr string) ([]Reaction, error)
	ReactionsCountByReactionable(reactionable Reactionable) ([]ReactionsCount, error)
	TranslateToCosmosMentions(body string) (string, error)
	TranslateToUsersMentions(body string) (string, error)
	InitialStakeBalanceByAddress(address string) (*InitialStakeBalance, error)
	OpenedClaimsSummary(date time.Time) ([]UserOpenedClaimsSummary, error)
	ClaimOfTheDayIDByCommunityID(communityID string) (int64, error)
	ClaimImageURL(claimID uint64) (string, error)
	VerifiedUserByID(id int64) (*User, error)
	GetAuthenticatedUser(identifier, password string) (*User, error)
	UserByID(ID int64) (*User, error)
	UserByEmailOrUsername(identifier string) (*User, error)
	UserByEmail(email string) (*User, error)
	UserByUsername(username string) (*User, error)
	UserByAddress(address string) (*User, error)
	UserByConnectedAccountTypeAndID(accountType, accountID string) (*User, error)
	IsTwitterUser(userID int64) bool
	InvitedUsers() ([]User, error)
	InvitedUsersByID(referrerID int64) ([]User, error)
	UnusedResetTokensByUser(userID int64) ([]PasswordResetToken, error)
	UnusedResetTokenByUserAndToken(userID int64, token string) (*PasswordResetToken, error)
	ConnectedAccountsByUserID(userID int64) ([]ConnectedAccount, error)
	ConnectedAccountByTypeAndID(accountType, accountID string) (*ConnectedAccount, error)
	UserProfileByAddress(addr string) (*UserProfile, error)
	UsersByAddress(addresses []string) ([]User, error)
	UserProfileByUsername(username string) (*UserProfile, error)
	ClaimViewsStats(date time.Time) ([]ClaimViewsStats, error)
	ClaimRepliesStats(date time.Time) ([]ClaimRepliesStats, error)
	Leaderboard(since time.Time, sortBy string, limit int, excludedCommunities []string) ([]LeaderboardTopUser, error)
	LastLeaderboardProcessedDate() (*LeaderboardProcessedDate, error)
	FeedLeaderboardInTransaction(fn func(*pg.Tx) error) error
	UpsertLeaderboardMetric(tx *pg.Tx, metric *LeaderboardUserMetric) error
	UpsertLeaderboardProcessedDate(tx *pg.Tx, metric *LeaderboardProcessedDate) error

	// deprecated, use UserProfileByAddress/UserProfileByUsername
	TwitterProfileByAddress(addr string) (*TwitterProfile, error)
	TwitterProfileByUsername(username string) (*TwitterProfile, error)
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
