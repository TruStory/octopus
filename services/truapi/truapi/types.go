package truapi

import (
	"time"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/bank"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tcmn "github.com/tendermint/tendermint/libs/common"

	"github.com/TruStory/octopus/services/truapi/db"
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
	ID           int64     `json:"id"`
	ClaimCreator string    `json:"claim_creator"`
	ClaimID      int64     `json:"claimId"`
	ArgumentID   int64     `json:"argumentId"`
	Creator      string    `json:"creator"`
	Timestamp    time.Time `json:"timestamp"`
}

// V2 Truchain structs

// AppAccount represents graphql serializable representation of a cosmos account
type AppAccount struct {
	Address       string
	AccountNumber uint64
	Coins         sdk.Coins
	Sequence      uint64
	Pubkey        tcmn.HexBytes
	SlashCount    uint
	IsJailed      bool
	JailEndTime   time.Time
	CreatedTime   time.Time
}

// EarnedCoin represents trusteak earned in each category
type EarnedCoin struct {
	sdk.Coin
	CommunityID string
}

// TransactionReference represents an entity referenced in a transaction
type TransactionReference struct {
	ReferenceID uint64 `graphql:"referenceId"`
	Type        TransactionReferenceType
	Title       string
	Body        string
}

// TransactionReferenceType defines the type of ReferenceID in a transaction
type TransactionReferenceType int8

// Types of reference
const (
	ReferenceNone TransactionReferenceType = iota
	ReferenceArgument
	ReferenceClaim
	ReferenceAppAccount
)

// TransactionTypeTitle defines user readable text for each transaction type
var TransactionTypeTitle = []string{
	bank.TransactionRegistration:             "Account Created",
	bank.TransactionBacking:                  "Wrote an Argument",
	bank.TransactionBackingReturned:          "Refund: Wrote an Argument",
	bank.TransactionChallenge:                "Wrote an Argument",
	bank.TransactionChallengeReturned:        "Refund: Wrote an Argument",
	bank.TransactionUpvote:                   "Agreed with %s",
	bank.TransactionUpvoteReturned:           "Refund: Agreed with %s",
	bank.TransactionInterestArgumentCreation: "Reward: Wrote an Argument",
	bank.TransactionInterestUpvoteReceived:   "Reward: Agree received from %s",
	bank.TransactionInterestUpvoteGiven:      "Reward: Agreed with %s",
	bank.TransactionRewardPayout:             "Reward: Invite a friend",
}

// CommunityIconImage contains regular and active icon images
type CommunityIconImage struct {
	Regular string
	Active  string
}

// Slash will be imported from truchain in the future
type Slash struct {
	ID          uint64
	StakeID     uint64
	Creator     sdk.AccAddress
	CreatedTime time.Time
}

// Settings contains application specific settings
type Settings struct {
	MinClaimLength    int      `json:"minClaimLength"`
	MaxClaimLength    int      `json:"maxClaimLength"`
	MinArgumentLength int      `json:"minArgumentLength"`
	MaxArgumentLength int      `json:"maxArgumentLength"`
	MinSummaryLength  int      `json:"minSummaryLength"`
	MaxSummaryLength  int      `json:"maxSummaryLength"`
	MinCommentLength  int64    `json:"minCommentLength"`
	MaxCommentLength  int64    `json:"maxCommentLength"`
	BlockIntervalTime int64    `json:"blockIntervalTime"`
	DefaultStake      sdk.Coin `json:"defaultStake"`
}

var NotificationIcons = map[db.NotificationType]string{
	db.NotificationEarnedStake: "earned_trustake.png",
	db.NotificationJailed:      "jailed.png",
	db.NotificationNotHelpful:  "not_helpful.png",
	db.NotificationSlashed:     "slashed.png",
	db.NotificationUnjailed:    "unjailed.png",
}
