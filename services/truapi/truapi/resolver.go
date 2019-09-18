package truapi

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/account"
	"github.com/TruStory/truchain/x/bank"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/slashing"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/julianshen/og"
	tcmn "github.com/tendermint/tendermint/libs/common"
	stripmd "github.com/writeas/go-strip-markdown"
)

type queryByCommunityID struct {
	CommunityID string `graphql:"communityId"`
}

type queryByClaimID struct {
	ID uint64 `graphql:"id"`
}

type queryByArgumentID struct {
	ID uint64 `graphql:"id"`
}

type queryByStakeID struct {
	ID uint64 `graphql:"id"`
}

type queryBySlashID struct {
	ID uint64 `graphql:"id"`
}

type queryByAddress struct {
	ID string `graphql:"id"`
}

type queryCommentsParams struct {
	ClaimID    *uint64 `graphql:"claimId"`
	ArgumentID *uint64 `graphql:"argumentId"`
	ElementID  *uint64 `graphql:"elementId"`
	// deprecated in favor of ClaimID
	ID uint64 `graphql:"id"`
}

type queryClaimArgumentParams struct {
	ClaimID uint64         `graphql:"id,optional"`
	Address *string        `graphql:"address,optional"`
	Filter  ArgumentFilter `graphql:"filter,optional"`
}

type queryByCommunityIDAndFeedFilter struct {
	CommunityID string     `graphql:"communityId,optional"`
	FeedFilter  FeedFilter `graphql:"feedFilter,optional"`
	IsSearch    bool       `graphql:"isSearch,optional"`
}

type queryReferredAppAccountsParams struct {
	Admin bool `graphql:"admin,optional"`
}

// claimMetricsBest represents all-time claim metrics
type claimMetricsBest struct {
	Claim                 claim.Claim
	TotalAmountStaked     sdk.Int
	TotalStakers          uint64
	TotalComments         int
	BackingChallengeDelta sdk.Int
}

// appAccountEarningsFilter is query params for filtering the app account's earnings
type appAccountEarningsFilter struct {
	ID   string `graphql:"id"`
	From string
	To   string
}

type appAccountCommunityEarning struct {
	Address      string   `json:"address"`
	CommunityID  string   `json:"community_id"`
	TotalEarned  sdk.Coin `json:"total_earned"`
	WeeklyEarned sdk.Coin `json:"weekly_earned"`
}

type appAccountEarning struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

type appAccountEarnings struct {
	NetEarnings sdk.Coin            `json:"net_earnings"`
	DataPoints  []appAccountEarning `json:"data_points"`
}

func (ta *TruAPI) appAccountsResolver(ctx context.Context, addresses []sdk.AccAddress) ([]*AppAccount, error) {
	queryRoute := path.Join(account.QuerierRoute, account.QueryPrimaryAccounts)
	res, err := ta.Query(queryRoute, account.QueryPrimaryAccountsParams{Addresses: addresses}, account.ModuleCodec)
	if err != nil {
		return nil, err
	}

	returnedAppAccounts := make([]account.PrimaryAccount, 0, len(addresses))
	err = account.ModuleCodec.UnmarshalJSON(res, &returnedAppAccounts)
	if err != nil {
		return nil, err
	}
	accounts := make([]*AppAccount, 0, len(addresses))
	for _, aa := range returnedAppAccounts {
		var pubKey []byte

		// GetPubKey can return nil and Bytes() will panic due to nil pointer
		if aa.GetPubKey() != nil {
			pubKey = aa.GetPubKey().Bytes()
		}

		accounts = append(accounts, &AppAccount{
			Address:       aa.GetAddress().String(),
			AccountNumber: aa.GetAccountNumber(),
			Coins:         aa.GetCoins(),
			Sequence:      aa.GetSequence(),
			Pubkey:        tcmn.HexBytes(pubKey),
			SlashCount:    uint(aa.SlashCount),
			IsJailed:      aa.IsJailed,
			JailEndTime:   aa.JailEndTime,
			CreatedTime:   aa.CreatedTime,
		})
	}
	return accounts, nil
}

func (ta *TruAPI) appAccountResolver(ctx context.Context, q queryByAddress) *AppAccount {
	l, ok := getDataLoaders(ctx)
	if !ok {
		fmt.Println("loaders not present")
		return nil
	}
	appAccount, err := l.appAccountLoader.Load(q.ID)
	if err != nil {
		return nil
	}
	return appAccount

}

// deprecated, use userProfileResolver instead
func (ta *TruAPI) twitterProfileResolver(ctx context.Context, addr string) db.TwitterProfile {
	twitterProfile, err := ta.DBClient.TwitterProfileByAddress(addr)
	if twitterProfile == nil {
		return db.TwitterProfile{}
	}
	if err != nil {
		// TODO [shanev]: Add back after adding error handling to resolvers
		// fmt.Println("Resolver err: ", err)
		return db.TwitterProfile{}
	}

	return *twitterProfile
}

func (ta *TruAPI) userProfileResolver(ctx context.Context, addr string) *db.UserProfile {
	loaders, ok := getDataLoaders(ctx)
	if !ok {
		return nil
	}
	profile, err := loaders.userProfileLoader.Load(addr)
	if err != nil {
		return nil
	}
	return profile
}

func (ta *TruAPI) userResolver(ctx context.Context, addr string) *db.User {
	user, err := ta.DBClient.UserByAddress(addr)
	if err != nil {
		fmt.Println("userResolver err: ", err)
		return nil
	}

	return user
}

func (ta *TruAPI) earnedBalanceResolver(ctx context.Context, q queryByAddress) sdk.Coin {
	earnedCoins := ta.earnedStakeResolver(ctx, q)
	balance := sdk.ZeroInt()
	for _, coin := range earnedCoins {
		balance = balance.Add(coin.Coin.Amount)
	}
	return sdk.NewCoin(app.StakeDenom, balance)
}

func (ta *TruAPI) earnedStakeResolver(ctx context.Context, q queryByAddress) []EarnedCoin {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("earnedStakeResolver err: ", err)
		return []EarnedCoin{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryEarnedCoins)
	res, err := ta.Query(queryRoute, staking.QueryEarnedCoinsParams{Address: address}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("earnedStakeResolver err: ", err)
		return []EarnedCoin{}
	}

	coins := new(sdk.Coins)
	err = staking.ModuleCodec.UnmarshalJSON(res, coins)
	if err != nil {
		fmt.Println("earnedCoin UnmarshalJSON err: ", err)
		return []EarnedCoin{}
	}

	communities := ta.communitiesResolver(ctx)

	earnedCoins := make([]EarnedCoin, 0)
	for _, community := range communities {
		earnedCoins = append(earnedCoins, EarnedCoin{
			sdk.NewCoin(community.ID, coins.AmountOf(community.ID)),
			community.ID,
		})
	}

	return earnedCoins
}

func (ta *TruAPI) pendingBalanceResolver(ctx context.Context, q queryByAddress) sdk.Coin {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("pendingBalanceResolver err: ", err)
		return sdk.Coin{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserStakes)
	res, err := ta.Query(queryRoute, staking.QueryUserStakesParams{Address: address}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("pendingBalanceResolver err: ", err)
		return sdk.Coin{}
	}

	stakes := make([]staking.Stake, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
	if err != nil {
		fmt.Println("stakes UnmarshalJSON err: ", err)
		return sdk.Coin{}
	}

	balance := sdk.NewCoin(app.StakeDenom, sdk.ZeroInt())
	for _, stake := range stakes {
		if !stake.Expired {
			balance = balance.Add(stake.Amount)
		}
	}

	return balance
}

func (ta *TruAPI) pendingStakeResolver(ctx context.Context, q queryByAddress) []EarnedCoin {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("pendingStakeResolver err: ", err)
		return []EarnedCoin{}
	}

	communities := ta.communitiesResolver(ctx)
	pendingStakes := make([]EarnedCoin, 0)

	for _, community := range communities {
		queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserCommunityStakes)
		res, err := ta.Query(queryRoute, staking.QueryUserCommunityStakesParams{Address: address, CommunityID: community.ID}, staking.ModuleCodec)
		if err != nil {
			fmt.Println("pendingStakeResolver err: ", err)
			return []EarnedCoin{}
		}

		stakes := make([]staking.Stake, 0)
		err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
		if err != nil {
			fmt.Println("stake UnmarshalJSON err: ", err)
			return []EarnedCoin{}
		}

		total := sdk.ZeroInt()
		for _, stake := range stakes {
			if !stake.Expired {
				total = total.Add(stake.Amount.Amount)
			}
		}

		pendingStakes = append(pendingStakes, EarnedCoin{
			sdk.Coin{
				Amount: total,
				Denom:  app.StakeDenom,
			},
			community.ID,
		})
	}

	return pendingStakes
}

func (ta *TruAPI) communitiesResolver(ctx context.Context) []community.Community {
	queryRoute := path.Join(community.QuerierRoute, community.QueryCommunities)
	res, err := ta.Query(queryRoute, struct{}{}, community.ModuleCodec)
	if err != nil {
		fmt.Println("communitiesResolver err: ", err)
		return []community.Community{}
	}

	cs := make([]community.Community, 0)
	err = community.ModuleCodec.UnmarshalJSON(res, &cs)
	if err != nil {
		fmt.Println("community UnmarshalJSON err: ", err)
		return []community.Community{}
	}

	// sort in alphabetical order
	sort.Slice(cs, func(i, j int) bool {
		return (cs)[j].Name > (cs)[i].Name
	})

	// exclude blacklisted communities
	filteredCommunities := make([]community.Community, 0)
	for _, c := range cs {
		if !contains(ta.APIContext.Config.Community.InactiveCommunities, c.ID) {
			filteredCommunities = append(filteredCommunities, c)
		}
	}

	return filteredCommunities
}

func (ta *TruAPI) communityResolver(ctx context.Context, q queryByCommunityID) *community.Community {
	queryRoute := path.Join(community.QuerierRoute, community.QueryCommunity)
	res, err := ta.Query(queryRoute, community.QueryCommunityParams{ID: q.CommunityID}, community.ModuleCodec)
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

func (ta *TruAPI) communityIconImageResolver(ctx context.Context, q community.Community) CommunityIconImage {
	return CommunityIconImage{
		Regular: joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_normal.png", q.ID)),
		Active:  joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("communities/%s_icon_active.png", q.ID)),
	}
}

func (ta *TruAPI) followedCommunityIDs(ctx context.Context) ([]string, error) {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return []string{}, errors.New("User not authenticated")
	}
	followedCommunities, err := ta.DBClient.FollowedCommunities(user.Address)
	if err != nil {
		return []string{}, err
	}
	followedCommunityIDs := make([]string, 0)
	for _, followedCommunity := range followedCommunities {
		followedCommunityIDs = append(followedCommunityIDs, followedCommunity.CommunityID)
	}
	return followedCommunityIDs, nil
}

func (ta *TruAPI) claimsResolver(ctx context.Context, q queryByCommunityIDAndFeedFilter) []claim.Claim {
	var res []byte
	var err error

	switch q.CommunityID {
	case "all":
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaims)
		res, err = ta.Query(queryRoute, struct{}{}, claim.ModuleCodec)
	case "home":
		communityIDs, cErr := ta.followedCommunityIDs(ctx)
		if cErr != nil {
			return []claim.Claim{}
		}
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryCommunitiesClaims)
		res, err = ta.Query(queryRoute, claim.QueryCommunitiesClaimsParams{CommunityIDs: communityIDs}, claim.ModuleCodec)
	default:
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryCommunityClaims)
		res, err = ta.Query(queryRoute, claim.QueryCommunityClaimsParams{CommunityID: q.CommunityID}, claim.ModuleCodec)
	}

	if err != nil {
		fmt.Println("claimsResolver err: ", err)
		return []claim.Claim{}
	}

	claims := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claims)
	if err != nil {
		panic(err)
	}

	if !q.IsSearch {
		claims = ta.removeClaimOfTheDay(claims, q.CommunityID)
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claims)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	filteredClaims := ta.filterFeedClaims(ctx, unflaggedClaims, q.FeedFilter)

	return filteredClaims
}

func (ta *TruAPI) claimResolver(ctx context.Context, q queryByClaimID) claim.Claim {
	queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaim)
	res, err := ta.Query(queryRoute, claim.QueryClaimParams{ID: q.ID}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("claimResolver err: ", err)
		return claim.Claim{}
	}

	c := new(claim.Claim)
	err = claim.ModuleCodec.UnmarshalJSON(res, c)
	if err != nil {
		fmt.Println("claim UnmarshalJSON err: ", err)
		return claim.Claim{}
	}

	return *c
}

func (ta *TruAPI) claimOfTheDayResolver(ctx context.Context, q queryByCommunityID) *claim.Claim {
	communityID := q.CommunityID
	// personal home feed and all claims feed should return same Claim of the Day
	if q.CommunityID == "home" {
		communityID = "all"
	}
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunityID(communityID)
	if err != nil {
		return nil
	}

	claim := ta.claimResolver(ctx, queryByClaimID{ID: uint64(claimOfTheDayID)})

	if claim.ID == 0 {
		return nil
	}

	return &claim
}

func (ta *TruAPI) removeClaimOfTheDay(claims []claim.Claim, communityID string) []claim.Claim {
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunityID(communityID)
	if err != nil {
		return claims
	}

	claimsWithoutClaimOfTheDay := make([]claim.Claim, 0)
	for _, claim := range claims {
		if claim.ID != uint64(claimOfTheDayID) {
			claimsWithoutClaimOfTheDay = append(claimsWithoutClaimOfTheDay, claim)
		}
	}

	return claimsWithoutClaimOfTheDay
}

func (ta *TruAPI) filterFlaggedClaims(claims []claim.Claim) ([]claim.Claim, error) {
	unflaggedClaims := make([]claim.Claim, 0)

	flaggedClaimsIDs, err := ta.DBClient.FlaggedStoriesIDs(ta.APIContext.Config.Flag.Admin, ta.APIContext.Config.Flag.Limit)
	if err != nil {
		return claims, err
	}

	for _, claim := range claims {
		if !containsInt64(flaggedClaimsIDs, int64(claim.ID)) {
			unflaggedClaims = append(unflaggedClaims, claim)
		}
	}

	return unflaggedClaims, nil
}

func (ta *TruAPI) claimArgumentsResolver(ctx context.Context, q queryClaimArgumentParams) []staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimArguments)
	res, err := ta.Query(queryRoute, staking.QueryClaimArgumentsParams{ClaimID: q.ClaimID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentsResolver err: ", err)
		return []staking.Argument{}
	}

	arguments := make([]staking.Argument, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &arguments)
	if err != nil {
		fmt.Println("[]staking.Argument UnmarshalJSON err: ", err)
		return []staking.Argument{}
	}
	filteredArguments := make([]staking.Argument, 0)
	for _, argument := range arguments {
		if q.Filter == ArgumentCreated {
			if argument.Creator.String() == *q.Address {
				filteredArguments = append(filteredArguments, argument)
			}
		} else if q.Filter == ArgumentAgreed {
			stakes := ta.claimArgumentStakesResolver(ctx, argument)
			for _, stake := range stakes {
				if stake.Creator.String() == *q.Address && stake.Type == staking.StakeUpvote {
					filteredArguments = append(filteredArguments, argument)
					break
				}
			}
		} else {
			filteredArguments = append(filteredArguments, argument)
		}
	}

	return filteredArguments
}

func (ta *TruAPI) claimArgumentResolver(ctx context.Context, q queryByArgumentID) *staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimArgument)
	res, err := ta.Query(queryRoute, staking.QueryClaimArgumentParams{ArgumentID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentResolver err: ", err)
		return nil
	}

	argument := new(staking.Argument)
	err = staking.ModuleCodec.UnmarshalJSON(res, argument)
	if err != nil {
		return nil
	}

	return argument
}

func (ta *TruAPI) topArgumentResolver(ctx context.Context, q claim.Claim) *staking.Argument {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimTopArgument)
	res, err := ta.Query(queryRoute, staking.QueryClaimTopArgumentParams{ClaimID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("topArgumentResolver err: ", err)
		return nil
	}

	argument := new(staking.Argument)
	err = staking.ModuleCodec.UnmarshalJSON(res, argument)
	if err != nil {
		fmt.Println("staking.Argument UnmarshalJSON err: ", err)
		return nil
	}

	// no top argument
	if argument.ID == 0 {
		return nil
	}

	return argument
}

// returns all argument writer and upvoter stakes on a claim
func (ta *TruAPI) claimStakesResolver(ctx context.Context, q claim.Claim) []staking.Stake {
	stakes := make([]staking.Stake, 0)
	arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: q.ID})
	for _, argument := range arguments {
		stakes = append(stakes, ta.claimArgumentStakesResolver(ctx, argument)...)
	}
	return stakes
}

func (ta *TruAPI) claimParticipantsResolver(ctx context.Context, q claim.Claim) []AppAccount {
	loaders, ok := getDataLoaders(ctx)
	if !ok {
		fmt.Println("loaders not present")
		return nil
	}
	stakes := ta.claimStakesResolver(ctx, q)
	comments, _ := ta.DBClient.CommentsByClaimID(q.ID)

	// use map to prevent duplicate participants
	participantsMap := make(map[string]string)
	for _, stake := range stakes {
		participantsMap[stake.Creator.String()] = stake.Creator.String()
	}
	for _, comment := range comments {
		participantsMap[comment.Creator] = comment.Creator
	}

	participants := make([]AppAccount, 0)

	addresses := make([]string, 0)
	for address := range participantsMap {
		addresses = append(addresses, address)
	}
	accounts, errs := loaders.appAccountLoader.LoadAll(addresses)
	errors := make([]error, 0)
	for _, e := range errs {
		if e != nil {
			errors = append(errors, e)
		}
	}
	if len(errors) > 0 {
		fmt.Println("errors", errors)
		return participants
	}
	for _, acc := range accounts {
		participants = append(participants, *acc)
	}
	return participants
}

func (ta *TruAPI) stakeResolver(ctx context.Context, q queryByStakeID) *staking.Stake {
	queryRoute := path.Join(staking.ModuleName, staking.QueryStake)
	res, err := ta.Query(queryRoute, staking.QueryStakeParams{StakeID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("stakeResolver err: ", err)
		return nil
	}

	stake := new(staking.Stake)
	err = staking.ModuleCodec.UnmarshalJSON(res, stake)
	if err != nil {
		return nil
	}

	return stake
}

func (ta *TruAPI) claimArgumentStakesResolver(ctx context.Context, q staking.Argument) []staking.Stake {
	queryRoute := path.Join(staking.ModuleName, staking.QueryArgumentStakes)
	res, err := ta.Query(queryRoute, staking.QueryArgumentStakesParams{ArgumentID: q.ID}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentStakesResolver err: ", err)
		return []staking.Stake{}
	}

	stakes := make([]staking.Stake, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
	if err != nil {
		fmt.Println("[]staking.Stake UnmarshalJSON err: ", err)
		return []staking.Stake{}
	}

	return stakes
}

func (ta *TruAPI) slashResolver(ctx context.Context, q queryBySlashID) *slashing.Slash {
	queryRoute := path.Join(slashing.ModuleName, slashing.QuerySlash)
	res, err := ta.Query(queryRoute, slashing.QuerySlashParams{ID: q.ID}, slashing.ModuleCodec)
	if err != nil {
		fmt.Println("slashResolver err: ", err)
		return nil
	}

	slash := new(slashing.Slash)
	err = slashing.ModuleCodec.UnmarshalJSON(res, slash)
	if err != nil {
		return nil
	}

	return slash
}

func (ta *TruAPI) slashesResolver(ctx context.Context) []slashing.Slash {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]slashing.Slash, 0)
	}

	settings := ta.settingsResolver(ctx)
	if !contains(settings.ClaimAdmins, user.Address) {
		return make([]slashing.Slash, 0)
	}

	queryRoute := path.Join(slashing.ModuleName, slashing.QuerySlashes)
	res, err := ta.Query(queryRoute, struct{}{}, slashing.ModuleCodec)
	if err != nil {
		fmt.Println("slashesResolver err: ", err)
		return nil
	}

	slashes := make([]slashing.Slash, 0)
	err = slashing.ModuleCodec.UnmarshalJSON(res, &slashes)
	if err != nil {
		fmt.Println("[]slashing.Slash UnmarshalJSON err: ", err)
		return nil
	}

	return slashes
}

func (ta *TruAPI) claimArgumentSlashesResolver(ctx context.Context, q staking.Argument) []slashing.Slash {
	queryRoute := path.Join(slashing.ModuleName, slashing.QueryArgumentSlashes)
	res, err := ta.Query(queryRoute, slashing.QueryArgumentSlashesParams{ArgumentID: q.ID}, slashing.ModuleCodec)
	if err != nil {
		fmt.Println("claimArgumentSlashesResolver err: ", err)
		return []slashing.Slash{}
	}

	slashes := make([]slashing.Slash, 0)
	err = slashing.ModuleCodec.UnmarshalJSON(res, &slashes)
	if err != nil {
		fmt.Println("[]slashing.Slash UnmarshalJSON err: ", err)
		return []slashing.Slash{}
	}

	return slashes
}

func (ta *TruAPI) claimArgumentUpvoteStakersResolver(ctx context.Context, q staking.Argument) []AppAccount {
	stakes := ta.claimArgumentStakesResolver(ctx, q)
	appAccounts := make([]AppAccount, 0)
	for _, stake := range stakes {
		if stake.Type == staking.StakeUpvote {
			appAccounts = append(appAccounts, *ta.appAccountResolver(ctx, queryByAddress{ID: stake.Creator.String()}))
		}
	}
	return appAccounts
}

func (ta *TruAPI) appAccountStakeResolver(ctx context.Context, q staking.Argument) *staking.Stake {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if ok {
		stakes := ta.claimArgumentStakesResolver(ctx, q)
		for _, stake := range stakes {
			if user.Address == stake.Creator.String() {
				return &stake
			}
		}
	}
	return nil
}

func (ta *TruAPI) appAccountSlashResolver(ctx context.Context, q staking.Argument) *slashing.Slash {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if ok {
		slashes := ta.claimArgumentSlashesResolver(ctx, q)
		for _, slash := range slashes {
			if user.Address == slash.Creator.String() {
				return &slash
			}
		}
	}
	return nil
}

func (ta *TruAPI) commentsResolver(ctx context.Context, q queryCommentsParams) []db.Comment {
	var comments []db.Comment
	var err error
	if q.ArgumentID == nil || q.ElementID == nil {
		id := q.ID
		if q.ClaimID != nil && *q.ClaimID > 0 {
			id = *q.ClaimID
		}
		comments, err = ta.DBClient.ClaimLevelComments(id)
		if err != nil {
			fmt.Println("commentsResolver err: ", err)
		}
	} else {
		comments, err = ta.DBClient.ArgumentLevelComments(*q.ArgumentID, *q.ElementID)
		if err != nil {
			fmt.Println("commentsResolver err: ", err)
		}
	}
	return comments
}

func (ta *TruAPI) claimQuestionsResolver(ctx context.Context, q queryByClaimID) []db.Question {
	questions, err := ta.DBClient.QuestionsByClaimID(q.ID)
	if err != nil {
		fmt.Println("claimQuestions err: ", err)
	}
	return questions
}

func (ta *TruAPI) appAccountClaimsCreatedResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("appAccountClaimsCreatedResolver err: ", err)
		return []claim.Claim{}
	}

	queryRoute := path.Join(claim.QuerierRoute, claim.QueryCreatorClaims)
	res, err := ta.Query(queryRoute, claim.QueryCreatorClaimsParams{Creator: creator}, claim.ModuleCodec)
	if err != nil {
		return []claim.Claim{}
	}

	claimsCreated := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claimsCreated)
	if err != nil {
		fmt.Println("[]claim.Claim UnmarshalJSON err: ", err)
		return []claim.Claim{}
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claimsCreated)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	return unflaggedClaims
}

func (ta *TruAPI) appAccountArgumentsResolver(ctx context.Context, q queryByAddress) []staking.Argument {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []staking.Argument{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserArguments)
	res, err := ta.Query(queryRoute, staking.QueryUserArgumentsParams{Address: creator}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountArguments err: ", err)
		return []staking.Argument{}
	}

	arguments := make([]staking.Argument, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &arguments)
	if err != nil {
		fmt.Println("[]staking.Argument UnmarshalJSON err: ", err)
		return []staking.Argument{}
	}

	return arguments
}

func (ta *TruAPI) appAccountClaimsWithArgumentsResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	arguments := ta.appAccountArgumentsResolver(ctx, q)

	// Use map to prevent duplicate claim IDs
	claimIDsWithArgumentMap := make(map[uint64]uint64)
	for _, argument := range arguments {
		claimIDsWithArgumentMap[argument.ClaimID] = argument.ClaimID
	}

	claimIDsWithArgument := make([]uint64, 0)
	for claimID := range claimIDsWithArgumentMap {
		claimIDsWithArgument = append(claimIDsWithArgument, claimID)
	}

	sort.Slice(claimIDsWithArgument, func(i int, j int) bool {
		return claimIDsWithArgument[i] > claimIDsWithArgument[j]
	})

	queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaimsByIDs)
	res, err := ta.Query(queryRoute, claim.QueryClaimsParams{IDs: claimIDsWithArgument}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountClaimsWithArguments err: ", err)
		return []claim.Claim{}
	}

	claimsWithArgument := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claimsWithArgument)
	if err != nil {
		fmt.Println("[]claim.Claim UnmarshalJSON err: ", err)
		return []claim.Claim{}
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claimsWithArgument)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	return unflaggedClaims
}

func (ta *TruAPI) appAccountClaimsWithAgreesResolver(ctx context.Context, q queryByAddress) []claim.Claim {
	stakes := ta.agreesResolver(ctx, q)

	// Use map to prevent duplicate claim IDs
	claimIDsWithAgreesMap := make(map[uint64]uint64)
	for _, stake := range stakes {
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{ID: stake.ArgumentID})
		if argument != nil {
			claimIDsWithAgreesMap[argument.ClaimID] = argument.ClaimID
		}
	}

	claimIDsWithAgrees := make([]uint64, 0)
	for claimID := range claimIDsWithAgreesMap {
		claimIDsWithAgrees = append(claimIDsWithAgrees, claimID)
	}

	sort.Slice(claimIDsWithAgrees, func(i int, j int) bool {
		return claimIDsWithAgrees[i] > claimIDsWithAgrees[j]
	})

	queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaimsByIDs)
	res, err := ta.Query(queryRoute, claim.QueryClaimsParams{IDs: claimIDsWithAgrees}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountClaimsWithAgrees err: ", err)
		return []claim.Claim{}
	}

	claimsWithAgrees := make([]claim.Claim, 0)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claimsWithAgrees)
	if err != nil {
		fmt.Println("[]claim.Claim UnmarshalJSON err: ", err)
		return []claim.Claim{}
	}

	unflaggedClaims, err := ta.filterFlaggedClaims(claimsWithAgrees)
	if err != nil {
		fmt.Println("filterFlaggedClaims err: ", err)
		panic(err)
	}

	return unflaggedClaims
}

func (ta *TruAPI) agreesResolver(ctx context.Context, q queryByAddress) []staking.Stake {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []staking.Stake{}
	}

	queryRoute := path.Join(staking.QuerierRoute, staking.QueryUserStakes)
	res, err := ta.Query(queryRoute, staking.QueryUserStakesParams{Address: creator}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("agreesResolver err: ", err)
		return []staking.Stake{}
	}

	stakes := make([]staking.Stake, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakes)
	if err != nil {
		return []staking.Stake{}
	}

	agrees := make([]staking.Stake, 0)
	for _, stake := range stakes {
		if stake.Type == staking.StakeUpvote {
			agrees = append(agrees, stake)
		}
	}

	return agrees
}

func (ta *TruAPI) appAccountTransactionsResolver(ctx context.Context, q queryByAddress) []bank.Transaction {
	creator, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		return []bank.Transaction{}
	}

	queryRoute := path.Join(bank.QuerierRoute, bank.QueryTransactionsByAddress)
	res, err := ta.Query(queryRoute, bank.QueryTransactionsByAddressParams{Address: creator}, bank.ModuleCodec)
	if err != nil {
		fmt.Println("appAccountTransactionsResolver err: ", err)
		return []bank.Transaction{}
	}

	transactions := make([]bank.Transaction, 0)
	err = bank.ModuleCodec.UnmarshalJSON(res, &transactions)
	if err != nil {
		return []bank.Transaction{}
	}

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[j].CreatedTime.Before(transactions[i].CreatedTime) && transactions[j].ID < transactions[i].ID
	})

	return transactions
}

func (ta *TruAPI) transactionReferenceResolver(ctx context.Context, t bank.Transaction) TransactionReference {
	var tr TransactionReference
	switch t.Type {
	case bank.TransactionCuratorReward:
		slash := ta.slashResolver(ctx, queryBySlashID{t.ReferenceID})
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{slash.ArgumentID})
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       TransactionTypeTitle[t.Type],
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionGift:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       TransactionTypeTitle[t.Type],
			Body:        "",
		}
	case bank.TransactionRewardPayout:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       TransactionTypeTitle[t.Type],
			Body:        "",
		}

	case bank.TransactionStakeCuratorSlashed:
		fallthrough
	case bank.TransactionStakeCreatorSlashed:
		fallthrough
	case bank.TransactionInterestUpvoteGivenSlashed:
		fallthrough
	case bank.TransactionInterestArgumentCreationSlashed:
		fallthrough
	case bank.TransactionInterestUpvoteReceivedSlashed:
		stake := ta.stakeResolver(ctx, queryByStakeID{t.ReferenceID})
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{stake.ArgumentID})
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       TransactionTypeTitle[t.Type],
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionUpvote:
		fallthrough
	case bank.TransactionUpvoteReturned:
		fallthrough
	case bank.TransactionInterestUpvoteGiven:
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{t.ReferenceID})
		creatorTwitterProfile := ta.twitterProfileResolver(ctx, argument.Creator.String())
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       fmt.Sprintf(TransactionTypeTitle[t.Type], creatorTwitterProfile.Username),
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionInterestUpvoteReceived:
		stake := ta.stakeResolver(ctx, queryByStakeID{ID: t.ReferenceID})
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{stake.ArgumentID})
		stakerTwitterProfile := ta.twitterProfileResolver(ctx, stake.Creator.String())
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       fmt.Sprintf(TransactionTypeTitle[t.Type], stakerTwitterProfile.Username),
			Body:        stripmd.Strip(argument.Summary),
		}
	case bank.TransactionBacking:
		fallthrough
	case bank.TransactionBackingReturned:
		fallthrough
	case bank.TransactionChallenge:
		fallthrough
	case bank.TransactionChallengeReturned:
		fallthrough
	case bank.TransactionInterestArgumentCreation:
		argument := ta.claimArgumentResolver(ctx, queryByArgumentID{t.ReferenceID})
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceArgument,
			Title:       TransactionTypeTitle[t.Type],
			Body:        stripmd.Strip(argument.Summary),
		}
	default:
		tr = TransactionReference{
			ReferenceID: t.ReferenceID,
			Type:        ReferenceNone,
			Title:       "",
			Body:        "",
		}
	}
	return tr
}

func (ta *TruAPI) claimImageResolver(ctx context.Context, q claim.Claim) string {
	claimImageURL, err := ta.DBClient.ClaimImageURL(q.ID)
	if err == nil && claimImageURL != "" {
		// found claimImageURL in the database, exit early
		return claimImageURL
	}

	n := (q.ID % 5) // random but deterministic placeholder image 0-4
	defaultImageURL := joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("claimImage_default_%d.png", n))

	if q.Source.String() == "" {
		claimImageURL = defaultImageURL
	} else {
		// fetch open graph image from source url website
		ogImage := og.OgImage{}
		err = og.GetPageDataFromUrl(q.Source.String(), &ogImage)

		if err != nil || ogImage.Url == "" {
			// no open graph image exists
			claimImageURL = defaultImageURL
		} else {
			claimImageURL = ogImage.Url
		}
	}

	_ = ta.DBClient.AddClaimImage(&db.ClaimImage{
		ClaimID:       q.ID,
		ClaimImageURL: claimImageURL,
	})

	return claimImageURL
}

func (ta *TruAPI) claimVideoResolver(ctx context.Context, q claim.Claim) *string {
	claimVideoURL, err := ta.DBClient.ClaimVideoURL(q.ID)
	if err == nil && claimVideoURL != "" {
		return &claimVideoURL
	}
	return nil
}

func (ta *TruAPI) settingsResolver(_ context.Context) Settings {
	queryRoute := path.Join(account.QuerierRoute, account.QueryParams)
	res, err := ta.Query(queryRoute, struct{}{}, account.ModuleCodec)
	if err != nil {
		fmt.Println("settingsResolver err: ", err)
		return Settings{}
	}

	accountParams := new(account.Params)
	err = account.ModuleCodec.UnmarshalJSON(res, &accountParams)
	if err != nil {
		fmt.Println("accountParams UnmarshalJSON err: ", err)
		return Settings{}
	}

	queryRoute = path.Join(claim.QuerierRoute, claim.QueryParams)
	res, err = ta.Query(queryRoute, struct{}{}, claim.ModuleCodec)
	if err != nil {
		fmt.Println("settingsResolver err: ", err)
		return Settings{}
	}

	claimParams := new(claim.Params)
	err = claim.ModuleCodec.UnmarshalJSON(res, &claimParams)
	if err != nil {
		fmt.Println("claimParams UnmarshalJSON err: ", err)
		return Settings{}
	}

	queryRoute = path.Join(staking.QuerierRoute, staking.QueryParams)
	res, err = ta.Query(queryRoute, struct{}{}, staking.ModuleCodec)
	if err != nil {
		fmt.Println("settingsResolver err: ", err)
		return Settings{}
	}

	stakingParams := new(staking.Params)
	err = staking.ModuleCodec.UnmarshalJSON(res, &stakingParams)
	if err != nil {
		fmt.Println("stakingParams UnmarshalJSON err: ", err)
		return Settings{}
	}

	queryRoute = path.Join(slashing.QuerierRoute, slashing.QueryParams)
	res, err = ta.Query(queryRoute, struct{}{}, slashing.ModuleCodec)
	if err != nil {
		fmt.Println("settingsResolver err: ", err)
		return Settings{}
	}

	slashingParams := new(slashing.Params)
	err = slashing.ModuleCodec.UnmarshalJSON(res, &slashingParams)
	if err != nil {
		fmt.Println("slashingParams UnmarshalJSON err: ", err)
		return Settings{}
	}

	creatorShare, err := strconv.ParseFloat(stakingParams.CreatorShare.String(), 64)
	if err != nil {
		return Settings{}
	}
	interestRate, err := strconv.ParseFloat(stakingParams.InterestRate.String(), 64)
	if err != nil {
		return Settings{}
	}
	curatorShare, err := strconv.ParseFloat(slashingParams.CuratorShare.String(), 64)
	if err != nil {
		return Settings{}
	}

	argumentCreationInterest := staking.Interest(stakingParams.InterestRate, stakingParams.ArgumentCreationStake, stakingParams.Period)
	upvoteInterest := staking.Interest(stakingParams.InterestRate, stakingParams.UpvoteStake, stakingParams.Period)
	upvoteCreatorReward := upvoteInterest.Mul(stakingParams.CreatorShare)
	upvoteStakerReward := upvoteInterest.Sub(upvoteCreatorReward)

	tomlParams := ta.APIContext.Config.Params
	return Settings{
		// account params
		Registrar:     accountParams.Registrar.String(),
		MaxSlashCount: int32(accountParams.MaxSlashCount),
		JailDuration:  strconv.FormatInt(accountParams.JailDuration.Nanoseconds(), 10),

		// claim params
		MinClaimLength: int32(claimParams.MinClaimLength),
		MaxClaimLength: int32(claimParams.MaxClaimLength),
		ClaimAdmins:    mapAccounts(claimParams.ClaimAdmins),

		// staking params
		Period:                   strconv.FormatInt(stakingParams.Period.Nanoseconds(), 10),
		ArgumentCreationStake:    stakingParams.ArgumentCreationStake,
		ArgumentBodyMinLength:    int32(stakingParams.ArgumentBodyMinLength),
		ArgumentBodyMaxLength:    int32(stakingParams.ArgumentBodyMaxLength),
		ArgumentSummaryMinLength: int32(stakingParams.ArgumentSummaryMinLength),
		ArgumentSummaryMaxLength: int32(stakingParams.ArgumentSummaryMaxLength),
		UpvoteStake:              stakingParams.UpvoteStake,
		CreatorShare:             creatorShare,
		InterestRate:             interestRate,
		StakingAdmins:            mapAccounts(stakingParams.StakingAdmins),
		MaxArgumentsPerClaim:     int32(stakingParams.MaxArgumentsPerClaim),
		ArgumentCreationReward:   sdk.NewCoin(app.StakeDenom, argumentCreationInterest.RoundInt()),
		UpvoteCreatorReward:      sdk.NewCoin(app.StakeDenom, upvoteCreatorReward.RoundInt()),
		UpvoteStakerReward:       sdk.NewCoin(app.StakeDenom, upvoteStakerReward.RoundInt()),

		// slashing params
		MinSlashCount:           int32(slashingParams.MinSlashCount),
		SlashMagnitude:          int32(slashingParams.SlashMagnitude),
		SlashMinStake:           slashingParams.SlashMinStake,
		SlashAdmins:             mapAccounts(slashingParams.SlashAdmins),
		CuratorShare:            curatorShare,
		MaxDetailedReasonLength: int32(slashingParams.MaxDetailedReasonLength),

		// off-chain params
		MinCommentLength:  int32(tomlParams.CommentMinLength),
		MaxCommentLength:  int32(tomlParams.CommentMaxLength),
		BlockIntervalTime: int32(tomlParams.BlockInterval),
		StakeDisplayDenom: db.CoinDisplayName,

		// deprecated
		MinArgumentLength: int32(stakingParams.ArgumentBodyMinLength),
		MaxArgumentLength: int32(stakingParams.ArgumentBodyMaxLength),
		MinSummaryLength:  int32(stakingParams.ArgumentSummaryMinLength),
		MaxSummaryLength:  int32(stakingParams.ArgumentSummaryMaxLength),
		DefaultStake:      sdk.NewCoin(app.StakeDenom, sdk.NewInt(30*app.Shanev)),
	}
}

func (ta *TruAPI) appAccountCommunityEarningsResolver(ctx context.Context, q queryByAddress) []appAccountCommunityEarning {
	now := time.Now()

	from := now.Add(-7 * 24 * time.Hour) // starting from 6 days before yesterday

	communityEarnings := make([]appAccountCommunityEarning, 0)
	communityWeeklyEarnings := make(map[string]sdk.Coin)
	communityAllTimeEarnings := make(map[string]sdk.Coin)

	// seeding empty communities
	communities := ta.communitiesResolver(ctx)
	for _, community := range communities {
		communityWeeklyEarnings[community.ID] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
		communityAllTimeEarnings[community.ID] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
	}

	transactions := ta.appAccountTransactionsResolver(ctx, q)
	// reversing the order of transactions
	for i := len(transactions)/2 - 1; i >= 0; i-- {
		opp := len(transactions) - 1 - i
		transactions[i], transactions[opp] = transactions[opp], transactions[i]
	}
	for _, transaction := range transactions {

		// Stake Earned
		if transaction.Type.OneOf([]bank.TransactionType{
			bank.TransactionInterestArgumentCreation,
			bank.TransactionInterestUpvoteReceived,
			bank.TransactionInterestUpvoteGiven,
			bank.TransactionRewardPayout,
		}) {
			// some transactions are in blacklisted communities so make sure to check the community exists in the map
			if _, ok := communityAllTimeEarnings[transaction.CommunityID]; ok {
				communityAllTimeEarnings[transaction.CommunityID] = communityAllTimeEarnings[transaction.CommunityID].Add(transaction.Amount)
			}
			if transaction.CreatedTime.After(from) {
				// some transactions are in blacklisted communities so make sure to check the community exists in the map
				if _, ok := communityWeeklyEarnings[transaction.CommunityID]; ok {
					communityWeeklyEarnings[transaction.CommunityID] = communityWeeklyEarnings[transaction.CommunityID].Add(transaction.Amount)
				}
			}
		}

		// Stake  Lost
		if transaction.Type.OneOf([]bank.TransactionType{
			bank.TransactionInterestArgumentCreationSlashed,
			bank.TransactionInterestUpvoteGivenSlashed,
			bank.TransactionInterestUpvoteReceivedSlashed,
		}) {
			// some transactions are in blacklisted communities so make sure to check the community exists in the map
			if _, ok := communityAllTimeEarnings[transaction.CommunityID]; ok {
				communityAllTimeEarnings[transaction.CommunityID] = communityAllTimeEarnings[transaction.CommunityID].Sub(transaction.Amount)
			}
			if transaction.CreatedTime.After(from) {
				// some transactions are in blacklisted communities so make sure to check the community exists in the map
				if _, ok := communityWeeklyEarnings[transaction.CommunityID]; ok {
					communityWeeklyEarnings[transaction.CommunityID] = communityWeeklyEarnings[transaction.CommunityID].Sub(transaction.Amount)
				}
			}
		}
	}

	for communityID := range communityAllTimeEarnings {
		communityEarnings = append(communityEarnings, appAccountCommunityEarning{
			Address:      q.ID,
			CommunityID:  communityID,
			WeeklyEarned: communityWeeklyEarnings[communityID],
			TotalEarned:  communityAllTimeEarnings[communityID],
		})
	}

	return communityEarnings
}

func (ta *TruAPI) appAccountEarningsResolver(ctx context.Context, q appAccountEarningsFilter) appAccountEarnings {
	now := time.Now()

	dataPoints := make([]appAccountEarning, 0)
	netEarnings := sdk.NewCoin(app.StakeDenom, sdk.ZeroInt())
	mappedDataPoints := make(map[string]sdk.Coin)
	var mappedSortedKeys []string
	// seeding empty dates
	from, err := time.Parse("2006-01-02", q.From)
	if err != nil {
		panic(err)
	}
	for date := from; date.Before(now); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		mappedDataPoints[key] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
		mappedSortedKeys = append(mappedSortedKeys, key) // storing the key so that we can later sort the map in the same order
	}

	transactions := ta.appAccountTransactionsResolver(ctx, queryByAddress{ID: q.ID})
	// reversing the order of transactions
	for i := len(transactions)/2 - 1; i >= 0; i-- {
		opp := len(transactions) - 1 - i
		transactions[i], transactions[opp] = transactions[opp], transactions[i]
	}

	runningBalance := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
	dailyRunningBalances := make(map[string]sdk.Coin)
	firstTxnDate := transactions[0].CreatedTime
	firstDataPointDate, err := time.Parse("2006-01-02", mappedSortedKeys[0])
	if err != nil {
		panic(err)
	}

	beginning := firstDataPointDate
	if firstTxnDate.Before(firstDataPointDate) {
		beginning = firstTxnDate
	}
	for date := beginning; date.Before(now); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		dailyRunningBalances[key] = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
	}

	for _, transaction := range transactions {
		key := transaction.CreatedTime.Format("2006-01-02")

		if transaction.Type.AllowedForDeduction() {
			runningBalance = runningBalance.Sub(transaction.Amount)
		} else {
			runningBalance = runningBalance.Add(transaction.Amount)
		}
		dailyRunningBalances[key] = runningBalance

		// Stake Earned
		if transaction.Type.OneOf([]bank.TransactionType{
			bank.TransactionInterestArgumentCreation,
			bank.TransactionInterestUpvoteReceived,
			bank.TransactionInterestUpvoteGiven,
			bank.TransactionRewardPayout,
		}) {
			if transaction.CreatedTime.After(from) {
				netEarnings = netEarnings.Add(transaction.Amount)
			}
		}

		// Stake  Lost
		if transaction.Type.OneOf([]bank.TransactionType{
			bank.TransactionInterestArgumentCreationSlashed,
			bank.TransactionInterestUpvoteGivenSlashed,
			bank.TransactionInterestUpvoteReceivedSlashed,
		}) {
			if transaction.CreatedTime.After(from) {
				netEarnings = netEarnings.Sub(transaction.Amount)
			}
		}
	}

	// seeding the dates that are missing in between
	runningBalance = sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
	for date := beginning; date.Before(now); date = date.AddDate(0, 0, 1) {
		key := date.Format("2006-01-02")
		if !dailyRunningBalances[key].IsZero() {
			runningBalance = dailyRunningBalances[key]
		}

		dailyRunningBalances[key] = runningBalance
	}

	for _, key := range mappedSortedKeys {
		dataPoints = append(dataPoints, appAccountEarning{
			Date:   key,
			Amount: dailyRunningBalances[key].Amount.Quo(sdk.NewInt(app.Shanev)).ToDec().RoundInt64(),
		})
	}

	// reduced data points
	var reducedDataPoints []appAccountEarning
	maximumDataPoints := 51
	if len(dataPoints) > maximumDataPoints {
		gap := len(dataPoints) / maximumDataPoints
		for i := 0; i < len(dataPoints); i++ {
			if i%gap == 0 { // removing every element at the nth-gap
				reducedDataPoints = append(reducedDataPoints, dataPoints[i])
			}
		}
	} else {
		reducedDataPoints = dataPoints
	}
	return appAccountEarnings{
		NetEarnings: netEarnings,
		DataPoints:  reducedDataPoints,
	}
}

func (ta *TruAPI) unreadNotificationsCountResolver(ctx context.Context, q struct{}) *db.NotificationsCountResponse {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return &db.NotificationsCountResponse{
			Count: 0,
		}
	}
	response, err := ta.DBClient.UnreadNotificationEventsCountByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return response
}

func (ta *TruAPI) unseenNotificationsCountResolver(ctx context.Context, q struct{}) *db.NotificationsCountResponse {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return &db.NotificationsCountResponse{
			Count: 0,
		}
	}
	response, err := ta.DBClient.UnseenNotificationEventsCountByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return response
}

func (ta *TruAPI) notificationsResolver(ctx context.Context, q struct{}) []db.NotificationEvent {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]db.NotificationEvent, 0)
	}
	evts, err := ta.DBClient.NotificationEventsByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return evts
}

func (ta *TruAPI) invitesResolver(ctx context.Context) []db.Invite {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]db.Invite, 0)
	}

	userProfile, err := ta.DBClient.UserProfileByAddress(user.Address)
	if err != nil {
		panic(err)
	}

	// TODO: pull this in from an ENV
	if strings.EqualFold(userProfile.Username, "lilrushishah") ||
		strings.EqualFold(userProfile.Username, "patel0phone") ||
		strings.EqualFold(userProfile.Username, "iam_preethi") ||
		strings.EqualFold(userProfile.Username, "truted2") ||
		strings.EqualFold(userProfile.Username, "mohitmamoria") ||
		strings.EqualFold(userProfile.Username, "shanev") {
		invites, err := ta.DBClient.Invites()
		if err != nil {
			panic(err)
		}
		return invites
	}
	invites, err := ta.DBClient.InvitesByAddress(user.Address)
	if err != nil {
		panic(err)
	}
	return invites
}

func (ta *TruAPI) referredAppAccountsResolver(ctx context.Context, q queryReferredAppAccountsParams) []AppAccount {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok {
		return make([]AppAccount, 0)
	}

	var users []db.User
	var err error
	settings := ta.settingsResolver(ctx)

	if q.Admin == true && contains(settings.ClaimAdmins, user.Address) {
		users, err = ta.DBClient.ReferredUsers()
		if err != nil {
			fmt.Println("referredAppAccountsResolver err: ", err)
			return make([]AppAccount, 0)
		}
	} else {
		users, err = ta.DBClient.ReferredUsersByID(user.ID)
		if err != nil {
			fmt.Println("referredAppAccountsResolver err: ", err)
			return make([]AppAccount, 0)
		}
	}

	appAccounts := make([]AppAccount, 0)
	for _, user := range users {
		if user.Address != "" {
			appAccounts = append(appAccounts, *ta.appAccountResolver(ctx, queryByAddress{ID: user.Address}))
		}
	}
	return appAccounts
}

func (ta *TruAPI) followsCommunity(ctx context.Context, q queryByCommunityID) bool {
	user, ok := ctx.Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return false
	}
	follows, err := ta.DBClient.FollowsCommunity(user.Address, q.CommunityID)
	if err != nil {
		return false
	}
	return follows
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func containsInt64(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func mapAccounts(accAddresses []sdk.AccAddress) []string {
	addressString := make([]string, len(accAddresses))
	for key, address := range accAddresses {
		addressString[key] = address.String()
	}
	return addressString
}
