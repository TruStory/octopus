package truapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"path"
	"time"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/bank/exported"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/community"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

type UserCommunityMetrics struct {
	Claims                  int
	Arguments               int
	AgreesGiven             int
	AgreesReceived          int
	Staked                  sdk.Coin
	StakedArgument          sdk.Coin
	StakedAgree             sdk.Coin
	InterestArgumentCreated sdk.Coin
	InterestAgreeReceived   sdk.Coin
	InterestAgreeGiven      sdk.Coin
	CuratorReward           sdk.Coin
	InterestSlashed         sdk.Coin
	StakeSlashed            sdk.Coin
	ClaimsOpened            int64
	UniqueClaimsOpened      int64
	EarnedCoin              sdk.Coin
	PendingStake            sdk.Coin
}

type UserMetrics struct {
	Balance          sdk.Coin
	CommunityMetrics map[string]*UserCommunityMetrics
}

type Metrics struct {
	UserMetrics map[string]*UserMetrics
}

func (m *Metrics) getUserMetrics(address string) *UserMetrics {
	userMetrics, ok := m.UserMetrics[address]
	if !ok {
		userMetrics = &UserMetrics{CommunityMetrics: make(map[string]*UserCommunityMetrics),
			Balance: sdk.NewInt64Coin(app.StakeDenom, 0)}
		m.UserMetrics[address] = userMetrics
	}
	return userMetrics
}
func (m *Metrics) getUserCommunityMetric(address, communityID string) *UserCommunityMetrics {
	userMetrics := m.getUserMetrics(address)
	ucm, ok := userMetrics.CommunityMetrics[communityID]
	if !ok {
		ucm = &UserCommunityMetrics{
			InterestArgumentCreated: sdk.NewInt64Coin(app.StakeDenom, 0),
			InterestAgreeReceived:   sdk.NewInt64Coin(app.StakeDenom, 0),
			InterestAgreeGiven:      sdk.NewInt64Coin(app.StakeDenom, 0),
			CuratorReward:           sdk.NewInt64Coin(app.StakeDenom, 0),
			InterestSlashed:         sdk.NewInt64Coin(app.StakeDenom, 0),
			StakeSlashed:            sdk.NewInt64Coin(app.StakeDenom, 0),
			EarnedCoin:              sdk.NewInt64Coin(app.StakeDenom, 0),
			Staked:                  sdk.NewInt64Coin(app.StakeDenom, 0),
			StakedArgument:          sdk.NewInt64Coin(app.StakeDenom, 0),
			StakedAgree:             sdk.NewInt64Coin(app.StakeDenom, 0),
			PendingStake:            sdk.NewInt64Coin(app.StakeDenom, 0),
		}
		userMetrics.CommunityMetrics[communityID] = ucm
	}
	return ucm
}

func (ta *TruAPI) getClaimArguments(claimID uint64) ([]staking.Argument, error) {
	queryRoute := path.Join(staking.ModuleName, staking.QueryClaimArguments)
	res, err := ta.Query(queryRoute, staking.QueryClaimArgumentsParams{ClaimID: claimID}, staking.ModuleCodec)
	if err != nil {
		return nil, err
	}

	arguments := make([]staking.Argument, 0)
	err = staking.ModuleCodec.UnmarshalJSON(res, &arguments)
	if err != nil {
		return nil, err
	}
	return arguments, nil
}

func notExpiredAt(date, created, end time.Time) bool {
	betaReleaseDate, err := time.Parse("2006-01-02", "2019-07-11")
	if err != nil {
		return false
	}
	betaReleaseDate = betaReleaseDate.UTC()

	// return always as expired any stake created before beta.
	if created.Before(betaReleaseDate) {
		return false
	}
	if date.Before(created) {
		return false
	}
	if date.After(end) {
		return false
	}
	if !created.Before(end) {
		return false
	}
	return true
}

func (ta *TruAPI) HandleUsersMetrics(w http.ResponseWriter, r *http.Request) {
	jobTime := time.Now().UTC().Format("200601021504")
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

	beforeDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get all claims
	claims := make([]claim.Claim, 0)
	result, err := ta.Query(
		path.Join(claim.QuerierRoute, claim.QueryClaimsBeforeTime),
		claim.QueryClaimsTimeParams{CreatedTime: beforeDate},
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

	// For each user, get the available stake calculated.
	users := make([]db.User, 0)
	err = ta.DBClient.FindAll(&users)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}

	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}
	chainMetrics := &Metrics{UserMetrics: make(map[string]*UserMetrics)}

	for _, claim := range claims {
		if !claim.CreatedTime.Before(beforeDate) {
			continue
		}
		argumentIDCreator := make(map[uint64]string)
		ucm := chainMetrics.getUserCommunityMetric(claim.Creator.String(), claim.CommunityID)
		ucm.Claims++
		arguments, err := ta.getClaimArguments(claim.ID)
		for _, argument := range arguments {
			if !argument.CreatedTime.Before(beforeDate) {
				continue
			}
			acm := chainMetrics.getUserCommunityMetric(argument.Creator.String(), claim.CommunityID)
			acm.Arguments++
			argumentIDCreator[argument.ID] = argument.Creator.String()
		}
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusInternalServerError)
		}
		stakes := ta.claimStakesResolver(r.Context(), claim)
		for _, stake := range stakes {
			if !stake.CreatedTime.Before(beforeDate) {
				continue
			}
			scm := chainMetrics.getUserCommunityMetric(stake.Creator.String(), claim.CommunityID)
			if !stake.Expired || notExpiredAt(beforeDate, stake.CreatedTime, stake.EndTime) {
				scm.PendingStake = scm.PendingStake.Add(stake.Amount)
			}
			if stake.Type == staking.StakeUpvote {
				scm.StakedAgree = scm.StakedAgree.Add(stake.Amount)
				chainMetrics.getUserCommunityMetric(argumentIDCreator[stake.ArgumentID], stake.CommunityID).AgreesReceived++
				scm.AgreesGiven++
			}

			if stake.Type != staking.StakeUpvote {
				scm.StakedArgument = scm.StakedArgument.Add(stake.Amount)
			}
			scm.Staked = scm.Staked.Add(stake.Amount)

		}
	}
	// Get all communities
	queryRoute := path.Join(community.QuerierRoute, community.QueryCommunities)
	res, err := ta.Query(queryRoute, struct{}{}, community.ModuleCodec)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}

	communities := make([]community.Community, 0)
	err = community.ModuleCodec.UnmarshalJSON(res, &communities)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}
	if len(communities) == 0 {
		render.Error(w, r, "no communities found", http.StatusInternalServerError)
		return
	}
	trackedTransactions := []exported.TransactionType{
		exported.TransactionBacking,
		exported.TransactionChallenge,
		exported.TransactionInterestArgumentCreation,
		exported.TransactionInterestUpvoteReceived,
		exported.TransactionInterestUpvoteGiven,
	}
	w.Header().Add("Content-Type", "text/csv")
	csvw := csv.NewWriter(w)
	header := []string{"job_date_time", "date", "address", "username", "balance",
		"community", "community_name", "stake_earned",
		"claims_created", "claims_opened", "unique_claims_opened",
		"arguments_created", "agrees_received", "agrees_given",
		"staked", "staked_arguments", "staked_agrees",
		"interest_argument_creation", "interest_agree_received", "interest_agree_given", "reward_not_helpful",
		"interest_slashed", "stake_slashed", "pending_stake",
	}
	err = csvw.Write(header)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}
	openedClaims, err := ta.DBClient.OpenedClaimsSummary(beforeDate)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
	}
	for _, userOpenedClaims := range openedClaims {
		userMetrics := chainMetrics.getUserCommunityMetric(userOpenedClaims.Address, userOpenedClaims.CommunityID)
		userMetrics.ClaimsOpened = userOpenedClaims.OpenedClaims
		userMetrics.UniqueClaimsOpened = userOpenedClaims.UniqueOpenedClaims
	}
	for _, user := range users {
		if !user.CreatedAt.Before(beforeDate) {
			continue
		}
		transactions := ta.appAccountTransactionsResolver(r.Context(), queryByAddress{ID: user.Address})
		balance := sdk.NewInt64Coin(app.StakeDenom, 0)
		for _, transaction := range transactions {
			if !transaction.CreatedTime.Before(beforeDate) {
				continue
			}

			if transaction.Type.AllowedForDeduction() {
				transaction.Amount.Amount = transaction.Amount.Amount.Neg()
			}
			balance = balance.Add(transaction.Amount)
			if !transaction.Type.OneOf(trackedTransactions) {
				continue
			}
			if transaction.CommunityID == "" {
				render.Error(w, r,
					fmt.Sprintf("transaction %s [%d] must contain community id",
						transaction.Type.String(), transaction.ID),
					http.StatusInternalServerError)
				return
			}

			ucm := chainMetrics.getUserCommunityMetric(user.Address, transaction.CommunityID)
			switch transaction.Type {
			case exported.TransactionInterestArgumentCreation:
				ucm.InterestArgumentCreated = ucm.InterestArgumentCreated.Add(transaction.Amount)
				ucm.EarnedCoin = sdk.NewCoin(transaction.CommunityID, ucm.EarnedCoin.Amount.Add(transaction.Amount.Amount))
			case exported.TransactionInterestUpvoteReceived:
				ucm.InterestAgreeReceived = ucm.InterestAgreeReceived.Add(transaction.Amount)
				ucm.EarnedCoin = sdk.NewCoin(transaction.CommunityID, ucm.EarnedCoin.Amount.Add(transaction.Amount.Amount))
			case exported.TransactionInterestUpvoteGiven:
				ucm.InterestAgreeGiven = ucm.InterestAgreeGiven.Add(transaction.Amount)
				ucm.EarnedCoin = sdk.NewCoin(transaction.CommunityID, ucm.EarnedCoin.Amount.Add(transaction.Amount.Amount))
			case exported.TransactionCuratorReward:
				ucm.CuratorReward = ucm.CuratorReward.Add(transaction.Amount)
				ucm.EarnedCoin = sdk.NewCoin(transaction.CommunityID, ucm.EarnedCoin.Amount.Add(transaction.Amount.Amount))
			}

		}
		// "job_time", "date", "address", "username", "balance"
		rowStart := []string{jobTime, beforeDate.Format(time.RFC3339Nano), user.Address, user.Username, balance.Amount.String()}

		for _, community := range communities {
			// 	"community", "community_name"
			record := append(rowStart, community.ID)
			record = append(record, community.Name)
			m := chainMetrics.getUserCommunityMetric(user.Address, community.ID)
			// "stake_earned"
			record = append(record, m.EarnedCoin.Amount.String())
			// "claims_created", "claims_opened", "unique_claims_opened",
			record = append(record, fmt.Sprintf("%d", m.Claims))
			record = append(record, fmt.Sprintf("%d", m.ClaimsOpened))
			record = append(record, fmt.Sprintf("%d", m.UniqueClaimsOpened))
			// "arguments_created", "agrees_received", "agrees_given",
			record = append(record, fmt.Sprintf("%d", m.Arguments))
			record = append(record, fmt.Sprintf("%d", m.AgreesReceived))
			record = append(record, fmt.Sprintf("%d", m.AgreesGiven))
			// "staked", "staked_argument", "staked_agree"
			record = append(record, m.Staked.Amount.String())
			record = append(record, m.StakedArgument.Amount.String())
			record = append(record, m.StakedAgree.Amount.String())
			// "interest_argument_creation", "interest_agree_received", "interest_agree_given", "reward_not_helpful",
			record = append(record, m.InterestArgumentCreated.Amount.String())
			record = append(record, m.InterestAgreeReceived.Amount.String())
			record = append(record, m.InterestAgreeGiven.Amount.String())
			record = append(record, fmt.Sprintf("%d", 0))
			// "interest_slashed", "stake_slashed", "at_stake"
			record = append(record, fmt.Sprintf("%d", 0))
			record = append(record, fmt.Sprintf("%d", 0))
			record = append(record, m.PendingStake.Amount.String())
			err = csvw.Write(record)
			if err != nil {
				render.Error(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		csvw.Flush()
	}

}
