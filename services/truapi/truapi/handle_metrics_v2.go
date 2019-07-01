package truapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
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
			Balance:          sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
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
	Username string   `json:"username"`
	Balance  sdk.Coin `json:"balance"`

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
	AvailableStake     sdk.Coin `json:"available_stake"`
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
				AvailableStake:     sdk.NewCoin(app.StakeDenom, sdk.NewInt(0)),
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

func (userMetrics *UserMetricsV2) addAvailableStake(communityID string, amount sdk.Coin) {
	m := userMetrics.getMetricsByCommunity(communityID).Metrics
	m.AvailableStake = m.AvailableStake.Add(amount)
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
	result, err := ta.Query("claims_before_time", claim.QueryClaimsTimeParams{CreatedTime: until}, claim.ModuleCodec)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = claim.ModuleCodec.UnmarshalJSON(result, claims)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all communities
	communities := make([]community.Community, 0)
	result, err = ta.Query("all", struct{}{}, community.ModuleCodec) // TODO: fix the community query string
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = community.ModuleCodec.UnmarshalJSON(result, communities)
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

	for i, claim := range claims {
		fmt.Println(systemMetrics, i, claim)

		// range over all the stakings

		// range over all the slashings
	}
}
