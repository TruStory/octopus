package truapi

import (
	"net/url"
	"time"

	app "github.com/TruStory/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tcmn "github.com/tendermint/tendermint/libs/common"
)

// CredArgument represents an argument that earned cred based on likes.
type CredArgument struct {
	ID        int64          `json:"id" graphql:"id" `
	StoryID   int64          `json:"storyId" graphql:"storyId"`
	Body      string         `json:"body"`
	Creator   sdk.AccAddress `json:"creator" `
	Timestamp app.Timestamp  `json:"timestamp"`
	Vote      bool           `json:"vote"`
	Amount    sdk.Coin       `json:"coin"`
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

// V2 Truchain structs

// AppAccount will be imported from truchain in the future
type AppAccount struct {
	BaseAccount

	EarnedStake []EarnedCoin
	SlashCount  int
	IsJailed    bool
	JailEndTime time.Time
	CreatedTime time.Time
}

// EarnedCoin will be imported from truchain in the future
type EarnedCoin struct {
	sdk.Coin

	CommunityID int64
}

// BaseAccount will be imported from truchain in the future
type BaseAccount struct {
	Address       string
	Coins         sdk.Coins
	PubKey        tcmn.HexBytes
	AccountNumber uint64
	Sequence      uint64
}

// Community will be imported from truchain in the future
type Community struct {
	ID               int64
	Name             string
	Slug             string
	Description      string
	TotalEarnedStake sdk.Coin
}

// CommunityIconImage contains regular and active icon images
type CommunityIconImage struct {
	Regular string
	Active  string
}

// Claim will be imported from truchain in the future
type Claim struct {
	ID              int64
	CommunityID     int64
	Body            string
	Creator         sdk.AccAddress
	Source          url.URL
	TotalBacked     sdk.Coin
	TotalChallenged sdk.Coin
	TotalStakers    int64
	CreatedTime     time.Time
}

// Argument will be imported from truchain in the future
type Argument struct {
	Stake

	ClaimID      int64
	Summary      string
	Body         string
	UpvotedCount int64
	UpvotedStake sdk.Coin
	SlashCount   int
	IsUnhelpful  bool
	UpdatedTime  time.Time
}

// Stake will be imported from truchain in the future
type Stake struct {
	ID          int64
	ArgumentID  int64
	Type        StakeType
	Stake       sdk.Coin
	Creator     sdk.AccAddress
	CreatedTime time.Time
	EndTime     time.Time
}

// StakeType will be imported from truchain in the future
type StakeType int

// will be imported from truchain in the future
const (
	Backing   StakeType = iota // 0
	Challenge                  // 1
	Upvote                     // 2
)

// Settings contains application specific settings
type Settings struct {
	MinClaimLength    int64    `json:"minClaimLength"`
	MaxClaimLength    int64    `json:"maxClaimLength"`
	MinArgumentLength int64    `json:"minArgumentLength"`
	MaxArgumentLength int64    `json:"maxArgumentLength"`
	MinSummaryLength  int64    `json:"minSummaryLength"`
	MaxSummaryLength  int64    `json:"maxSummaryLength"`
	MinCommentLength  int64    `json:"minCommentLength"`
	MaxCommentLength  int64    `json:"maxCommentLength"`
	BlockIntervalTime int64    `json:"blockIntervalTime"`
	DefaultStake      sdk.Coin `json:"defaultStake"`
}

// ClaimComment contains claim level comments
type ClaimComment struct {
	ID         int64
	ParentID   int64
	ArgumentID int64
	Body       string
	Creator    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}
