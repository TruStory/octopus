package truapi

import (
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/account"
	"github.com/TruStory/truchain/x/bank"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/julianshen/og"
	tcmn "github.com/tendermint/tendermint/libs/common"
	stripmd "github.com/writeas/go-strip-markdown"
)

// ArgumentFilter defines filters for claimArguments
type ArgumentFilter int64

// List of ArgumentFilter types
const (
	ArgumentAll ArgumentFilter = iota
	ArgumentCreated
	ArgumentAgreed
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

type queryByAddress struct {
	ID string `graphql:"id"`
}

type queryClaimArgumentParams struct {
	ClaimID uint64         `graphql:"id,optional"`
	Address *string        `graphql:"address,optional"`
	Filter  ArgumentFilter `graphql:"filter,optional"`
}

type queryByCommunityIDAndFeedFilter struct {
	CommunityID string     `graphql:"communityId,optional"`
	FeedFilter  FeedFilter `graphql:"feedFilter,optional"`
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
	TotalArguments int
	TotalComments  int
	TotalStakes    int64
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

func (ta *TruAPI) appAccountResolver(ctx context.Context, q queryByAddress) *AppAccount {
	address, err := sdk.AccAddressFromBech32(q.ID)
	if err != nil {
		fmt.Println("account AccAddressFromBech32 err: ", err)
		return nil
	}

	queryRoute := path.Join(account.QuerierRoute, account.QueryAppAccount)
	res, err := ta.Query(queryRoute, account.QueryAppAccountParams{Address: address}, account.ModuleCodec)
	if err != nil {
		return nil
	}

	var aa = new(account.AppAccount)
	err = account.ModuleCodec.UnmarshalJSON(res, aa)
	if err != nil {
		fmt.Println("AppAccount UnmarshalJSON err: ", err)
		return nil
	}

	var pubKey []byte

	// GetPubKey can return nil and Bytes() will panic due to nil pointer
	if aa.GetPubKey() != nil {
		pubKey = aa.GetPubKey().Bytes()
	}

	return &AppAccount{
		Address:       aa.GetAddress().String(),
		AccountNumber: aa.GetAccountNumber(),
		Coins:         aa.GetCoins(),
		Sequence:      aa.GetSequence(),
		Pubkey:        tcmn.HexBytes(pubKey),
		SlashCount:    aa.SlashCount,
		IsJailed:      aa.IsJailed,
		JailEndTime:   aa.JailEndTime,
		CreatedTime:   aa.CreatedTime,
	}
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

	fmt.Println("inactive", ta.APIContext.Config.Community.InactiveCommunities)

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

func (ta *TruAPI) claimsResolver(ctx context.Context, q queryByCommunityIDAndFeedFilter) []claim.Claim {
	var res []byte
	var err error
	if q.CommunityID == "all" {
		queryRoute := path.Join(claim.QuerierRoute, claim.QueryClaims)
		res, err = ta.Query(queryRoute, struct{}{}, claim.ModuleCodec)
	} else {
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

	claimsWithoutClaimOfTheDay := ta.removeClaimOfTheDay(claims, q.CommunityID)

	unflaggedClaims, err := ta.filterFlaggedClaims(claimsWithoutClaimOfTheDay)
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
	claimOfTheDayID, err := ta.DBClient.ClaimOfTheDayIDByCommunityID(q.CommunityID)
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
	stakes := ta.claimStakesResolver(ctx, q)
	comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: q.ID})

	// use map to prevent duplicate participants
	participantsMap := make(map[string]string)
	for _, stake := range stakes {
		participantsMap[stake.Creator.String()] = stake.Creator.String()
	}
	for _, comment := range comments {
		participantsMap[comment.Creator] = comment.Creator
	}

	participants := make([]AppAccount, 0)
	for address := range participantsMap {
		participants = append(participants, *ta.appAccountResolver(ctx, queryByAddress{ID: address}))
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

func (ta *TruAPI) claimCommentsResolver(ctx context.Context, q queryByClaimID) []db.Comment {
	comments, err := ta.DBClient.CommentsByClaimID(q.ID)
	if err != nil {
		panic(err)
	}
	return comments
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
		return transactions[j].CreatedTime.Before(transactions[i].CreatedTime)
	})

	return transactions
}

func (ta *TruAPI) transactionReferenceResolver(ctx context.Context, t bank.Transaction) TransactionReference {
	var tr TransactionReference
	switch t.Type {
	case bank.TransactionRegistration:
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

func (ta *TruAPI) sourceURLPreviewResolver(ctx context.Context, q claim.Claim) string {
	sourceURLPreview, err := ta.DBClient.ClaimSourceURLPreview(q.ID)
	if err == nil && sourceURLPreview != "" {
		// found sourceURLPreview in the database, exit early
		return sourceURLPreview
	}

	n := (q.ID % 5) // random but deterministic placeholder image 0-4
	defaultPreview := joinPath(ta.APIContext.Config.App.S3AssetsURL, fmt.Sprintf("sourceUrlPreview_default_%d.png", n))

	if q.Source.String() == "" {
		sourceURLPreview = defaultPreview
	} else {
		// fetch open graph image from source url website
		ogImage := og.OgImage{}
		err = og.GetPageDataFromUrl(q.Source.String(), &ogImage)

		if err != nil || ogImage.Url == "" {
			// no open graph image exists
			sourceURLPreview = defaultPreview
		} else {
			sourceURLPreview = ogImage.Url
		}
	}

	_ = ta.DBClient.AddClaimSourceURLPreview(&db.ClaimSourceURLPreview{
		ClaimID:          q.ID,
		SourceURLPreview: sourceURLPreview,
	})

	return sourceURLPreview
}

func (ta *TruAPI) settingsResolver(ctx context.Context) Settings {
	params := ta.paramsResolver(ctx)
	tomlParams := ta.APIContext.Config.Params
	return Settings{
		MinClaimLength:    params.ClaimParams.MinClaimLength,
		MaxClaimLength:    params.ClaimParams.MaxClaimLength,
		MinArgumentLength: params.StakingParams.ArgumentBodyMinLength,
		MaxArgumentLength: params.StakingParams.ArgumentBodyMaxLength,
		MinSummaryLength:  params.StakingParams.ArgumentSummaryMinLength,
		MaxSummaryLength:  params.StakingParams.ArgumentSummaryMaxLength,
		MinCommentLength:  tomlParams.CommentMinLength,
		MaxCommentLength:  tomlParams.CommentMaxLength,
		BlockIntervalTime: tomlParams.BlockInterval,
		DefaultStake:      sdk.NewCoin(app.StakeDenom, sdk.NewInt(tomlParams.DefaultStake*app.Shanev)),
	}
}

func (ta *TruAPI) filterFeedClaims(ctx context.Context, claims []claim.Claim, filter FeedFilter) []claim.Claim {
	if filter == Latest {
		// Reverse chronological order
		return claims
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
		// highest volume of activity in last 72 hours
		// # of new arguments
		// # of new agree stakes    TODO: for now its only stakes created on arguments written in the last 72 hours
		// # of new comments
		metrics := make([]claimMetricsTrending, 0)
		for _, claim := range claims {
			comments := ta.claimCommentsResolver(ctx, queryByClaimID{ID: claim.ID})
			recentComments := 0
			totalComments := 0
			for _, comment := range comments {
				totalComments++
				if comment.CreatedAt.After(time.Now().AddDate(0, 0, -3)) {
					recentComments++
				}
			}
			arguments := ta.claimArgumentsResolver(ctx, queryClaimArgumentParams{ClaimID: claim.ID})
			recentArguments := 0
			totalArguments := 0
			var totalStakes int64
			for _, argument := range arguments {
				totalArguments++
				if argument.CreatedTime.After(time.Now().AddDate(0, 0, -3)) {
					recentArguments++
					totalStakes += argument.TotalStake.Amount.Int64()
				}
			}

			if totalArguments+totalComments > 0 {
				metric := claimMetricsTrending{
					Claim:          claim,
					TotalComments:  recentComments,
					TotalArguments: recentArguments,
					TotalStakes:    totalStakes,
				}
				metrics = append(metrics, metric)
			}
		}
		sort.Slice(metrics, func(i, j int) bool {
			if metrics[i].TotalArguments > metrics[j].TotalArguments {
				return true
			}
			if metrics[i].TotalArguments < metrics[j].TotalArguments {
				return false
			}
			if metrics[i].TotalStakes > metrics[j].TotalStakes {
				return true
			}
			if metrics[i].TotalStakes < metrics[j].TotalStakes {
				return false
			}
			if metrics[i].TotalComments > metrics[j].TotalComments {
				return true
			}
			if metrics[i].TotalComments < metrics[j].TotalComments {
				return false
			}
			return metrics[j].Claim.CreatedTime.Before(metrics[i].Claim.CreatedTime)
		})
		trendingClaims := make([]claim.Claim, 0)
		for _, metric := range metrics {
			trendingClaims = append(trendingClaims, metric.Claim)
		}
		return trendingClaims
	}
	return claims
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
