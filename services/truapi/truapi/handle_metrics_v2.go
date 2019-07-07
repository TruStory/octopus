package truapi

import (
	"net/http"
	"path"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SystemMetrics represents metrics across the system
type SystemMetrics struct {
	Users map[string]*UserMetricsV2 `json:"users"`
}

func (sysMetrics *SystemMetrics) getUserMetrics(address string) *UserMetricsV2 {
	userMetrics, ok := sysMetrics.Users[address]
	if !ok {
		userMetrics = &UserMetricsV2{
			AvailableStake:   sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
			CommunityMetrics: make(map[string]*CommunityMetrics),
		}
	}
	sysMetrics.setUserMetrics(address, userMetrics)
	return userMetrics
}

func (sysMetrics *SystemMetrics) setUserMetrics(address string, userMetrics *UserMetricsV2) {
	sysMetrics.Users[address] = userMetrics
}

// UserMetricsV2 a summary of different metrics per user
type UserMetricsV2 struct {
	// Per user
	AvailableStake sdk.Coin `json:"available_stake"`

	// For each community
	CommunityMetrics map[string]*CommunityMetrics `json:"community_metrics"`
}

// CommunityMetrics summary of metrics by community
type CommunityMetrics struct {
	CommunityID string     `json:"community_id"`
	Metrics     *MetricsV2 `json:"metrics"`
}

// MetricsV2 defines the numbers that are tracked
type MetricsV2 struct {
	// StakeBased Metrics
	TotalAmountStaked  sdk.Coin `json:"total_amount_staked"`
	StakeEarned        sdk.Coin `json:"stake_earned"`
	StakeLost          sdk.Coin `json:"stake_lost"`
	TotalAmountAtStake sdk.Coin `json:"total_amount_at_stake"`
}

func (userMetrics *UserMetricsV2) getMetricsByCommunity(communityID string) *CommunityMetrics {
	cm, ok := userMetrics.CommunityMetrics[communityID]
	if !ok {
		communityMetrics := &CommunityMetrics{
			CommunityID: communityID,
			Metrics: &MetricsV2{
				TotalAmountStaked:  sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
				StakeEarned:        sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
				StakeLost:          sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
				TotalAmountAtStake: sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
			},
		}
		userMetrics.CommunityMetrics[communityID] = communityMetrics
		return communityMetrics
	}
	return cm
}

func (userMetrics *UserMetricsV2) addAmoutStaked(communityID string, amount sdk.Coin) {
	m := userMetrics.getMetricsByCommunity(communityID).Metrics
	m.TotalAmountStaked = m.TotalAmountStaked.Add(amount)
}

func (userMetrics *UserMetricsV2) addStakeEarned(communityID string, amount sdk.Coin) {
	m := userMetrics.getMetricsByCommunity(communityID).Metrics
	m.StakeEarned = m.StakeEarned.Add(amount)
}

func (userMetrics *UserMetricsV2) addStakeLost(communityID string, amount sdk.Coin) {
	m := userMetrics.getMetricsByCommunity(communityID).Metrics
	m.StakeLost = m.StakeLost.Add(amount)
}

func (userMetrics *UserMetricsV2) addAmoutAtStake(communityID string, amount sdk.Coin) {
	m := userMetrics.getMetricsByCommunity(communityID).Metrics
	m.TotalAmountAtStake = m.TotalAmountAtStake.Add(amount)
}

func (userMetrics *UserMetricsV2) addAvailableStake(amount sdk.Coin) {
	userMetrics.AvailableStake = userMetrics.AvailableStake.Add(amount)
}

// HandleMetricsV2 dumps system wide metrics
func (ta *TruAPI) HandleMetricsV2(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	date := r.FormValue("date")
	if date == "" {
		render.Error(w, r, "provide a valid date", http.StatusBadRequest)
		return
	}

	until, err := time.Parse("2006-01-02", date)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all claims
	claims := make([]claim.Claim, 0)
	result, err := ta.Query(
		path.Join(claim.QuerierRoute, claim.QueryClaimsBeforeTime),
		claim.QueryClaimsTimeParams{CreatedTime: until},
		claim.ModuleCodec,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = claim.ModuleCodec.UnmarshalJSON(result, &claims)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all communities
	communities := make([]community.Community, 0)
	result, err = ta.Query(
		path.Join(community.QuerierRoute, community.QueryCommunities),
		struct{}{},
		community.ModuleCodec,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = community.ModuleCodec.UnmarshalJSON(result, &communities)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(communities) == 0 {
		render.Error(w, r, "no communities found", http.StatusInternalServerError)
		return
	}

	systemMetrics := &SystemMetrics{
		Users: make(map[string]*UserMetricsV2),
	}

	// For each user, get the available stake calculated.
	users := make([]db.TwitterProfile, 0)
	err = ta.DBClient.FindAll(&users)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}

	for _, user := range users {
		userMetrics := systemMetrics.getUserMetrics(user.Address)

		transactions := ta.appAccountTransactionsResolver(r.Context(), queryByAddress{ID: user.Address})
		for _, transaction := range transactions {
			if !transaction.CreatedTime.Before(until) {
				continue
			}

			userMetrics.addAvailableStake(transaction.Amount)
		}
	}

	// Calculate the community-specific metrics here
	for _, claim := range claims {
		// range over all the stakings
		arguments := ta.claimArgumentsResolver(r.Context(), queryClaimArgumentParams{ClaimID: claim.ID})
		for _, argument := range arguments {
			if !argument.CreatedTime.Before(until) {
				continue
			}

			totalBackingStakes := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))
			totalChallengingStakes := sdk.NewCoin(app.StakeDenom, sdk.NewInt(0))

			stakes := ta.claimArgumentStakesResolver(r.Context(), argument)
			for _, stake := range stakes {
				stakerMetrics := systemMetrics.getUserMetrics(stake.Creator.String())
				stakerMetrics.addAmoutStaked(claim.CommunityID, stake.Amount)

				if stake.Type == staking.StakeBacking {
					totalBackingStakes = totalBackingStakes.Add(stake.Amount)
				} else if stake.Type == staking.StakeChallenge {
					totalChallengingStakes = totalChallengingStakes.Add(stake.Amount)
				}

				// if the argument is still running
				if stake.EndTime.After(time.Now()) {
					stakerMetrics.addAmoutAtStake(claim.CommunityID, stake.Amount)
				}
			}

			for _, stake := range stakes {
				if totalBackingStakes.IsLT(totalChallengingStakes) && stake.Type == staking.StakeChallenge {
					// if backers lost.. but the stakes were of challenge, then earned
					systemMetrics.getUserMetrics(stake.Creator.String()).addStakeEarned(claim.CommunityID, stake.Amount)
				} else if totalChallengingStakes.IsLT(totalBackingStakes) && stake.Type == staking.StakeBacking {
					// if challengers lost.. but the stakes were of backing, then earned
					systemMetrics.getUserMetrics(stake.Creator.String()).addStakeEarned(claim.CommunityID, stake.Amount)
				}
			}
		}

		// TODO: addStakeLost method to be implemented when slashing module gets in
		systemMetrics.getUserMetrics(claim.Creator.String()).addStakeLost(claim.CommunityID, sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)))
	}


	render.JSON(w, r, systemMetrics, http.StatusOK)
}
