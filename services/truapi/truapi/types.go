package truapi

import (
	"time"

	"github.com/TruStory/truchain/x/bank"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tcmn "github.com/tendermint/tendermint/libs/common"

	"github.com/TruStory/octopus/services/truapi/db"
)

// FeedFilter is parameter for filtering the story feed
type FeedFilter int64

// List of filter types
const (
	None FeedFilter = iota
	Trending
	Latest
	Completed
	Best
)

// ArgumentFilter defines filters for claimArguments
type ArgumentFilter int64

// List of ArgumentFilter types
const (
	ArgumentAll ArgumentFilter = iota
	ArgumentCreated
	ArgumentAgreed
)

type LeaderboardMetricFilter int64
type LeaderboardDateFilter int64

const (
	LeaderboardMetricFilterTruEarned LeaderboardMetricFilter = iota
	LeaderboardMetricFilterAgreesReceived
	LeaderboardMetricFilterAgreesGiven
)

const (
	LeaderboardDateFilterLastWeek LeaderboardDateFilter = iota
	LeaderboardDateFilterLastMonth
	LeaderboardDateFilterLastYear
	LeaderboardDateFilterAllTime
)

var LeaderboardMetricSortByMapping = []string{
	LeaderboardMetricFilterTruEarned:      "earned",
	LeaderboardMetricFilterAgreesReceived: "agrees_received",
	LeaderboardMetricFilterAgreesGiven:    "agrees_given",
}

func (f LeaderboardMetricFilter) Value() string {
	// if unknown fallback to last tru earned
	if int(f) >= len(LeaderboardMetricSortByMapping) {
		return LeaderboardMetricSortByMapping[LeaderboardMetricFilterTruEarned]
	}
	return LeaderboardMetricSortByMapping[f]
}

var LeaderboardDateRangeMapping = []time.Duration{
	LeaderboardDateFilterLastWeek:  time.Duration(-1) * time.Hour * 24 * 7,
	LeaderboardDateFilterLastMonth: time.Duration(-1) * time.Hour * 24 * 30,
	LeaderboardDateFilterLastYear:  time.Duration(-1) * time.Hour * 24 * 365,
	LeaderboardDateFilterAllTime:   time.Duration(0),
}

func (f LeaderboardDateFilter) Value() time.Duration {
	// if unknown fallback to last week
	if int(f) >= len(LeaderboardDateRangeMapping) {
		return LeaderboardDateRangeMapping[LeaderboardDateFilterLastWeek]
	}
	return LeaderboardDateRangeMapping[f]
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

// EarnedCoin represents TRU earned in each category
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
	bank.TransactionBacking:                         "Wrote an Argument",
	bank.TransactionBackingReturned:                 "Refund: Wrote an Argument",
	bank.TransactionChallenge:                       "Wrote an Argument",
	bank.TransactionChallengeReturned:               "Refund: Wrote an Argument",
	bank.TransactionUpvote:                          "Agreed with %s",
	bank.TransactionUpvoteReturned:                  "Refund: Agreed with %s",
	bank.TransactionInterestArgumentCreation:        "Reward: Wrote an Argument",
	bank.TransactionCuratorReward:                   "Reward: Marked an Argument as not Helpful",
	bank.TransactionInterestUpvoteReceived:          "Reward: Agree received from %s",
	bank.TransactionInterestUpvoteGiven:             "Reward: Agreed with %s",
	bank.TransactionRewardPayout:                    "Reward: Invite a friend",
	bank.TransactionGift:                            "Gift",
	bank.TransactionStakeCuratorSlashed:             "Slash: Stake Agree slashed",
	bank.TransactionStakeCreatorSlashed:             "Slash: Stake Argument slashed",
	bank.TransactionInterestUpvoteGivenSlashed:      "Slash: Interest Agree Given slashed",
	bank.TransactionInterestArgumentCreationSlashed: "Slash: Interest Argument Created slashed",
	bank.TransactionInterestUpvoteReceivedSlashed:   "Slash: Interest Agree Received slashed",
}

// CommunityIconImage contains regular and active icon images
type CommunityIconImage struct {
	Regular string
	Active  string
}

// Settings contains application specific settings
type Settings struct {
	// account params
	Registrar     string
	MaxSlashCount int32
	JailDuration  string

	// claim params
	MinClaimLength int32
	MaxClaimLength int32
	ClaimAdmins    []string

	// staking params
	Period                   string
	ArgumentCreationStake    sdk.Coin
	ArgumentBodyMinLength    int32
	ArgumentBodyMaxLength    int32
	ArgumentSummaryMinLength int32
	ArgumentSummaryMaxLength int32
	UpvoteStake              sdk.Coin
	CreatorShare             float64
	InterestRate             float64
	StakingAdmins            []string
	MaxArgumentsPerClaim     int32
	ArgumentCreationReward   sdk.Coin
	UpvoteCreatorReward      sdk.Coin
	UpvoteStakerReward       sdk.Coin

	// slashing params
	MinSlashCount           int32
	SlashMinStake           sdk.Coin
	SlashMagnitude          int32
	SlashAdmins             []string
	CuratorShare            float64
	MaxDetailedReasonLength int32

	// off-chain params
	MinCommentLength  int32
	MaxCommentLength  int32
	BlockIntervalTime int32
	StakeDisplayDenom string

	// deprecated
	MinArgumentLength int32
	MaxArgumentLength int32
	MinSummaryLength  int32
	MaxSummaryLength  int32
	DefaultStake      sdk.Coin
}

var NotificationIcons = map[db.NotificationType]string{
	db.NotificationEarnedStake: "earned_trustake.png",
	db.NotificationJailed:      "jailed.png",
	db.NotificationNotHelpful:  "not_helpful.png",
	db.NotificationSlashed:     "slashed.png",
	db.NotificationUnjailed:    "unjailed.png",
}
