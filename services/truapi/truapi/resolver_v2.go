package truapi

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/argument"
	"github.com/TruStory/truchain/x/backing"
	"github.com/TruStory/truchain/x/challenge"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/users"
	sdk "github.com/cosmos/cosmos-sdk/types"
	amino "github.com/tendermint/go-amino"
)

// ArgumentFilter defines filters for claimArguments
type ArgumentFilter int64

// List of ArgumentFilter types
const (
	ArgumentAll ArgumentFilter = iota
	ArgumentCreated
	ArgumentAgreed
)

type queryByID struct {
	ID uint64 `json:"id"`
}

type queryByCommunityID struct {
	CommunityID uint64 `json:"community_id"`
}

type queryByCommunitySlug struct {
	CommunitySlug string `graphql:"communitySlug"`
}

type queryByClaimID struct {
	ID uint64 `graphql:"id"`
}

type queryByArgumentID struct {
	ID uint64 `graphql:"id"`
}

type queryByAddress struct {
	ID string `graphql:"id"`
}

type queryClaimArgumentParams struct {
	ClaimID uint64         `graphql:"id,optional"`
	Address *string        `graphql:"address,optional"`
	Filter  ArgumentFilter `graphql:"filter,optional"`
}

type queryByCommunitySlugAndFeedFilter struct {
	CommunitySlug string     `graphql:",optional"`
	FeedFilter    FeedFilter `graphql:",optional"`
}

type argumentMeta struct {
	Vote         bool
	UpvotedCount uint64
}

// claimMetricsBest represents all-time claim metrics
type claimMetricsBest struct {
	Claim                 claim.Claim
	TotalAmountStaked     sdk.Int
	TotalStakers          uint64
	TotalComments         int
	BackingChallengeDelta sdk.Int
}

// claimMetricsTrending represents claim metrics within last 24 hours
type claimMetricsTrending struct {
	Claim          claim.Claim
	TotalArguments int64
	TotalComments  int
	TotalStakes    int64
}

// SummaryLength is amount of characters allowed when summarizing an argument
const SummaryLength = 140

func convertStoryArgumentToClaimArgument(storyArgument argument.Argument, argumentMeta argumentMeta) Argument {
	bodyLength := len(storyArgument.Body)
	if bodyLength > SummaryLength {
		bodyLength = SummaryLength
	}
	summary := storyArgument.Body[:bodyLength]
	var stakeType StakeType
	if argumentMeta.Vote {
		stakeType = Backing
	} else {
		stakeType = Challenge
	}
	claimArgument := Argument{
		Stake: Stake{
			ID:          uint64(storyArgument.ID),
			Creator:     storyArgument.Creator,
			CreatedTime: storyArgument.Timestamp.CreatedTime,
			Type:        stakeType,
		},
		ClaimID:      uint64(storyArgument.StoryID),
		UpvotedCount: argumentMeta.UpvotedCount,
		Body:         storyArgument.Body,
		Summary:      summary,
	}
	return claimArgument
}

func convertCommentToClaimComment(comment db.Comment) ClaimComment {
	return ClaimComment{
		ID:         comment.ID,
		ParentID:   comment.ParentID,
		ArgumentID: comment.ArgumentID,
		Body:       comment.Body,
		Creator:    comment.Creator,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
		DeletedAt:  comment.DeletedAt,
	}
}

func convertBackingToStake(backing backing.Backing) Stake {
	return Stake{
		ID:          uint64(backing.ID()),
		ArgumentID:  uint64(backing.ArgumentID),
		Type:        Upvote,
		Stake:       backing.Amount(),
		Creator:     backing.Creator(),
		CreatedTime: backing.Timestamp().CreatedTime,
		EndTime:     backing.Timestamp().UpdatedTime,
	}
}

func convertChallengeToStake(challenge challenge.Challenge) Stake {
	return Stake{
		ID:          uint64(challenge.ID()),
		ArgumentID:  uint64(challenge.ArgumentID),
		Type:        Upvote,
		Stake:       challenge.Amount(),
		Creator:     challenge.Creator(),
		CreatedTime: challenge.Timestamp().CreatedTime,
		EndTime:     challenge.Timestamp().UpdatedTime,
	}
}

func (ta *TruAPI) appAccountResolver(ctx context.Context, q queryByAddress) AppAccount {
	addresses := users.QueryUsersByAddressesParams{
		Addresses: []string{q.ID},
	}

	res, err := ta.RunQuery("users/addresses", addresses)
	if err != nil {
		return AppAccount{}
	}

	users := new([]users.User)
	err = amino.UnmarshalJSON(res, users)
	if err != nil {
		return AppAccount{}
	}
	if len(*users) == 0 {
		return AppAccount{}
	}
	u := (*users)[0]

	// split User.Coins into AppAccount.Coins and AppAccount.EarnedStake
	trustake := make(sdk.Coins, 0)
	earnedStake := make([]EarnedCoin, 0)

	trustake = append(trustake, sdk.NewCoin(app.StakeDenom, u.Coins.AmountOf(app.StakeDenom)))

	communityID := int64(1)
	for _, coin := range u.Coins {
		if coin.Denom != app.StakeDenom {
			earnedCoin := sdk.NewCoin(app.StakeDenom, coin.Amount)
			earned := EarnedCoin{
				Coin:        earnedCoin,
				CommunityID: uint64(communityID),
			}
			earnedStake = append(earnedStake, earned)
			communityID++
		}
	}

	appAccount := AppAccount{
		BaseAccount: BaseAccount{
			Address:       u.Address,
			AccountNumber: u.AccountNumber,
			Coins:         trustake,
			Sequence:      u.Sequence,
			PubKey:        u.Pubkey,
		},
		EarnedStake: earnedStake,
	}

	return appAccount
}

func (ta *TruAPI) communitiesResolver(ctx context.Context) []community.Community {
	res, err := ta.Query("community/all", struct{}{}, community.ModuleCodec)
	if err != nil {
		fmt.Println("communitiesResolver err: ", err)
		return []community.Community{}
	}

	cs := new([]community.Community)
	err = community.ModuleCodec.UnmarshalJSON(res, cs)
	if err != nil {
		fmt.Println("community UnmarshalJSON err: ", err)
		return []community.Community{}
	}

	// sort in alphabetical order
	sort.Slice(*cs, func(i, j int) bool {
		return (*cs)[j].Name > (*cs)[i].Name
	})

	return *cs
}

func (ta *TruAPI) communityResolver(ctx context.Context, q queryByCommunitySlug) *community.Community {
	community, err := ta.getCommunityBySlug(ctx, q.CommunitySlug)
	if err != nil {
		return nil
	}
	return &community
}

func (ta *TruAPI) communityIconImageResolver(ctx context.Context, q community.Community) CommunityIconImage {
	return CommunityIconImage{
		Regular: joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_normal.png", q.Slug)),
		Active:  joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_active.png", q.Slug)),
	}
}

func (ta *TruAPI) claimsResolver(ctx context.Context, q queryByCommunitySlugAndFeedFilter) []claim.Claim {
	var res []byte
	var err error
	var community community.Community
	if q.CommunitySlug == "all" {
		res, err = ta.Query("claim/claims", struct{}{}, claim.ModuleCodec)
	} else {
		community, err = ta.getCommunityBySlug(ctx, q.CommunitySlug)
		if err != nil {
			return []claim.Claim{}
		}
		res, err = ta.Query("claim/community_claims", queryByCommunityID{CommunityID: community.ID}, claim.ModuleCodec)
	}
	if err != nil {
		fmt.Println("claimsResolver err: ", err)
		return []claim.Claim{}
	}

	claims := new([]claim.Claim)
	err = claim.ModuleCodec.UnmarshalJSON(res, claims)
	if err != nil {
		panic(err)
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(*claims)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	filteredClaims := ta.filterFeedClaims(ctx, unflaggedClaims, q.FeedFilter)

	return filteredClaims
}

func (ta *TruAPI) claimResolver(ctx context.Context, q queryByClaimID) claim.Claim {
	res, err := ta.Query("claim/claim", claim.QueryClaimParams{ID: q.ID}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("claimResolver err: ", err)
		return claim.Claim{}
	}

	var c claim.Claim
	err = claim.ModuleCodec.UnmarshalJSON(res, &c)
	if err != nil {
		fmt.Println("claim UnmarshalJSON err: ", err)
		return claim.Claim{}
	}

	return c
}

func (ta *TruAPI) claimOfTheDayResolver(ctx context.Context, q queryByCommunitySlug) *claim.Claim {
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunitySlug(q.CommunitySlug)
	if err != nil {
		return nil
	}

	claim := ta.claimResolver(ctx, queryByClaimID{ID: uint64(claimOfTheDayID)})
	return &claim
}

func (ta *TruAPI) filterFlaggedClaims(claims []claim.Claim) ([]claim.Claim, error) {
	unflaggedClaims := make([]claim.Claim, 0)
	for _, claim := range claims {
		claimFlags, err := ta.DBClient.FlaggedStoriesByStoryID(int64(claim.ID))
		if err != nil {
			return nil, err
		}
		if len(claimFlags) > 0 {
			if claimFlags[0].Creator == ta.APIContext.Config.Flag.Admin {
				continue
			}
		}
		if len(claimFlags) < ta.APIContext.Config.Flag.Limit {
			unflaggedClaims = append(unflaggedClaims, claim)
		}
	}

	return unflaggedClaims, nil
}

func (ta *TruAPI) getCommunityBySlug(ctx context.Context, slug string) (community.Community, error) {
	// client pages require all claims to live under a community
	// "all" is a community for the homepage which shows all the claims
	if slug == "all" {
		return community.Community{
			ID:   0,
			Slug: "all",
			Name: "All",
		}, nil
	}
	communities := ta.communitiesResolver(ctx)
	var comm community.Community
	for _, community := range communities {
		if community.Slug == slug {
			comm = community
			break
		}
	}
	if comm.ID == 0 {
		return community.Community{}, errors.New("Category not found")
	}

	return comm, nil
}

func (ta *TruAPI) getCommunityByID(ctx context.Context, q queryByID) *community.Community {
	res, err := ta.Query("community/id", community.QueryCommunityParams{ID: q.ID}, community.ModuleCodec)
	if err != nil {
		fmt.Println("getCommunityByIDResolver err: ", err)
		return nil
	}

	c := new(community.Community)
	err = community.ModuleCodec.UnmarshalJSON(res, c)
	if err != nil {
		return nil
	}
	return c
}

func (ta *TruAPI) claimArgumentsResolver(ctx context.Context, q queryClaimArgumentParams) []Argument {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: int64(q.ClaimID)})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: int64(q.ClaimID)})
	storyArguments := map[int64]*argumentMeta{}
	for _, backing := range backings {
		if storyArguments[backing.ArgumentID] == nil {
			storyArguments[backing.ArgumentID] = &argumentMeta{
				Vote:         backing.VoteChoice(),
				UpvotedCount: 0,
			}
		} else {
			storyArguments[backing.ArgumentID].UpvotedCount++
		}
	}
	for _, challenge := range challenges {
		if storyArguments[challenge.ArgumentID] == nil {
			storyArguments[challenge.ArgumentID] = &argumentMeta{
				Vote:         challenge.VoteChoice(),
				UpvotedCount: 0,
			}
		} else {
			storyArguments[challenge.ArgumentID].UpvotedCount++
		}
	}
	claimArguments := make([]Argument, 0)
	for argumentID, argumentMeta := range storyArguments {
		storyArgument := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: argumentID})
		claimArgument := convertStoryArgumentToClaimArgument(storyArgument, *argumentMeta)
		if q.Filter == ArgumentCreated {
			if *q.Address == claimArgument.Creator.String() {
				claimArguments = append(claimArguments, claimArgument)
			}
		} else if q.Filter == ArgumentAgreed {
			stakers := ta.claimArgumentStakersResolver(ctx, claimArgument)
			for _, staker := range stakers {
				if *q.Address == staker.Address && staker.Address != claimArgument.Creator.String() {
					claimArguments = append(claimArguments, claimArgument)
					break
				}
			}
		} else {
			claimArguments = append(claimArguments, claimArgument)
		}
	}

	return claimArguments
}

func (ta *TruAPI) topArgumentResolver(ctx context.Context, q claim.Claim) *Argument {
	arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: q.ID})
	if len(arguments) == 0 {
		return nil
	}
	return &arguments[0]
}

func (ta *TruAPI) claimStakersResolver(ctx context.Context, q claim.Claim) []AppAccount {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: int64(q.ID)})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: int64(q.ID)})
	appAccounts := make([]AppAccount, 0)
	for _, backing := range backings {
		appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: backing.Creator().String()}))
	}
	for _, challenge := range challenges {
		appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: challenge.Creator().String()}))
	}
	return appAccounts
}

func (ta *TruAPI) claimParticipantsResolver(ctx context.Context, q claim.Claim) []AppAccount {
	participants := ta.claimStakersResolver(ctx, q)
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: q.ID})
	for _, comment := range comments {
		if !participantExists(participants, comment.Creator) {
			participants = append(participants, ta.appAccountResolver(ctx, queryByAddress{ID: comment.Creator}))
		}
	}
	if !participantExists(participants, q.Creator.String()) {
		participants = append(participants, ta.appAccountResolver(ctx, queryByAddress{ID: q.Creator.String()}))
	}
	return participants
}

func participantExists(participants []AppAccount, participantToAdd string) bool {
	for _, participant := range participants {
		if participantToAdd == participant.Address {
			return true
		}
	}
	return false
}

func (ta *TruAPI) claimArgumentStakesResolver(ctx context.Context, q Argument) []Stake {
	backings := ta.backingsResolver(ctx, app.QueryByIDParams{ID: int64(q.ClaimID)})
	challenges := ta.challengesResolver(ctx, app.QueryByIDParams{ID: int64(q.ClaimID)})
	stakes := make([]Stake, 0)
	for _, backing := range backings {
		if uint64(backing.ArgumentID) == q.ID && !backing.Creator().Equals(q.Creator) {
			stakes = append(stakes, convertBackingToStake(backing))
		}
	}
	for _, challenge := range challenges {
		if uint64(challenge.ArgumentID) == q.ID && !challenge.Creator().Equals(q.Creator) {
			stakes = append(stakes, convertChallengeToStake(challenge))
		}
	}
	return stakes
}

func (ta *TruAPI) claimArgumentStakersResolver(ctx context.Context, q Argument) []AppAccount {
	stakes := ta.claimArgumentStakesResolver(ctx, q)
	appAccounts := make([]AppAccount, 0)
	for _, stake := range stakes {
		appAccounts = append(appAccounts, ta.appAccountResolver(ctx, queryByAddress{ID: stake.Creator.String()}))
	}
	return appAccounts
}

func (ta *TruAPI) appAccountStakeResolver(ctx context.Context, q Argument) *Stake {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if ok {
		if user.Address == q.Creator.String() {
			return &q.Stake
		}
		stakes := ta.claimArgumentStakesResolver(ctx, q)
		for _, stake := range stakes {
			if user.Address == stake.Creator.String() {
				return &stake
			}
		}
	}
	return nil
}

func (ta *TruAPI) claimCommentsResolver(ctx context.Context, q queryByClaimID) []ClaimComment {
	arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: q.ID})
	comments := make([]db.Comment, 0)
	for _, argument := range arguments {
		argument := ta.argumentResolver(ctx, app.QueryArgumentByID{ID: int64(argument.ID)})
		argComments := ta.commentsResolver(ctx, argument)
		comments = append(comments, argComments...)
	}
	claimComments := make([]ClaimComment, 0)
	for _, comment := range comments {
		claimComments = append(claimComments, convertCommentToClaimComment(comment))
	}
	return claimComments
}

func (ta *TruAPI) stakesResolver(_ context.Context, q queryByArgumentID) []Stake {
	return []Stake{}
}

func (ta *TruAPI) appAccountClaimsCreatedResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	allClaims := ta.claimsResolver(ctx, queryByCommunitySlugAndFeedFilter{CommunitySlug: "all"})
	claimsCreated := make([]claim.Claim, 0)
	for _, claim := range allClaims {
		if claim.Creator.String() == q.ID {
			claimsCreated = append(claimsCreated, claim)
		}
	}
	return claimsCreated
}

func (ta *TruAPI) appAccountClaimsWithArgumentsResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	allClaims := ta.claimsResolver(ctx, queryByCommunitySlugAndFeedFilter{CommunitySlug: "all"})
	claimsWithArguments := make([]claim.Claim, 0)
	for _, claim := range allClaims {
		arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: claim.ID, Address: &q.ID, Filter: ArgumentCreated})
		if len(arguments) > 0 {
			claimsWithArguments = append(claimsWithArguments, claim)
		}
	}
	return claimsWithArguments
}

func (ta *TruAPI) appAccountClaimsWithAgreesResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	allClaims := ta.claimsResolver(ctx, queryByCommunitySlugAndFeedFilter{CommunitySlug: "all"})
	claimsWithAgrees := make([]claim.Claim, 0)
	for _, claim := range allClaims {
		arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: claim.ID, Address: &q.ID, Filter: ArgumentAgreed})
		if len(arguments) > 0 {
			claimsWithAgrees = append(claimsWithAgrees, claim)
		}
	}
	return claimsWithAgrees
}

func (ta *TruAPI) settingsResolver(_ context.Context) Settings {
	return Settings{
		MinClaimLength:    25,
		MaxClaimLength:    140,
		MinArgumentLength: 25,
		MaxArgumentLength: 1250,
		MinSummaryLength:  25,
		MaxSummaryLength:  140,
		MinCommentLength:  5,
		MaxCommentLength:  1000,
		BlockIntervalTime: 5000,
		DefaultStake:      sdk.NewCoin(app.StakeDenom, sdk.NewInt(30*app.Shanev)),
	}
}

func (ta *TruAPI) filterFeedClaims(ctx context.Context, claims []claim.Claim, filter FeedFilter) []claim.Claim {
	if filter == Latest {
		// Reverse chronological order, up to 1 week
		latestClaims := make([]claim.Claim, 0)
		for _, claim := range claims {
			if claim.CreatedTime.After(time.Now().AddDate(0, 0, -7)) {
				latestClaims = append(latestClaims, claim)
			}
		}
		sort.Slice(latestClaims, func(i, j int) bool {
			return latestClaims[j].CreatedTime.Before(latestClaims[i].CreatedTime)
		})
		return latestClaims
	} else if filter == Best {
		// Total amount staked
		// Total stakers
		// Total comments
		// Smallest delta between Backing vs Challenge stake
		metrics := make([]claimMetricsBest, 0)
		for _, claim := range claims {
			totalAmountStaked := claim.TotalBacked.Add(claim.TotalChallenged).Amount
			totalStakers := claim.TotalStakers
			totalComments := len(ta.claimCommentsResolver(ctx, queryByClaimID{ID: claim.ID}))
			var backingChallengeDelta sdk.Int
			if claim.TotalBacked.IsGTE(claim.TotalChallenged) {
				backingChallengeDelta = claim.TotalBacked.Sub(claim.TotalChallenged).Amount
			} else {
				backingChallengeDelta = claim.TotalChallenged.Sub(claim.TotalBacked).Amount
			}
			metric := claimMetricsBest{
				Claim:                 claim,
				TotalAmountStaked:     totalAmountStaked,
				TotalStakers:          totalStakers,
				TotalComments:         totalComments,
				BackingChallengeDelta: backingChallengeDelta,
			}
			metrics = append(metrics, metric)
		}
		sort.Slice(metrics, func(i, j int) bool {
			if metrics[i].TotalAmountStaked.GT(metrics[j].TotalAmountStaked) {
				return true
			}
			if metrics[i].TotalAmountStaked.LT(metrics[j].TotalAmountStaked) {
				return false
			}
			if metrics[i].TotalStakers > metrics[j].TotalStakers {
				return true
			}
			if metrics[i].TotalStakers < metrics[j].TotalStakers {
				return false
			}
			if metrics[i].TotalComments > metrics[j].TotalComments {
				return true
			}
			if metrics[i].TotalComments < metrics[j].TotalComments {
				return false
			}
			if metrics[i].BackingChallengeDelta.GT(metrics[j].BackingChallengeDelta) {
				return true
			}
			return false
		})
		bestClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			bestClaims = append(bestClaims, metric.Claim)
		}
		return bestClaims
	} else if filter == Trending {
		// highest volume of activity in last 24 hours
		// # of new arguments       TODO: need tendermint tags
		// # of new comments
		// # of new agree stakes    TODO: need tendermint tags
		metrics := make([]claimMetricsTrending, 0)
		for _, claim := range claims {
			comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: claim.ID})
			totalComments := 0
			for _, comment := range comments {
				if comment.CreatedAt.Before(time.Now().AddDate(0, 0, -1)) {
					totalComments++
				}
			}
			metric := claimMetricsTrending{
				Claim:         claim,
				TotalComments: totalComments,
			}
			metrics = append(metrics, metric)
		}
		sort.Slice(metrics, func(i, j int) bool {
			if metrics[i].TotalArguments > metrics[j].TotalArguments {
				return true
			}
			if metrics[i].TotalArguments < metrics[j].TotalArguments {
				return false
			}
			if metrics[i].TotalComments > metrics[j].TotalComments {
				return true
			}
			if metrics[i].TotalComments < metrics[j].TotalComments {
				return false
			}
			return metrics[j].TotalStakes-metrics[i].TotalStakes > 0
		})
		trendingClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			trendingClaims = append(trendingClaims, metric.Claim)
		}
		return trendingClaims
	}
	return claims
}
