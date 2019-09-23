package truapi

import (
	"fmt"
	"net/http"

	"github.com/go-pg/pg"

	"github.com/TruStory/octopus/services/truapi/truapi/render"

	"github.com/TruStory/octopus/services/truapi/db"
)

type metricUser struct {
	ID               int64                `json:"id"`
	ReferrerUsername string               `json:"referrer_username"`
	ReferredEmail    string               `json:"referrer_email"`
	FullName         string               `json:"full_name"`
	Username         string               `json:"username"`
	Email            string               `json:"email"`
	InvitesLeft      int64                `json:"invites_left"`
	JourneyCompleted []db.UserJourneyStep `json:"journey_completed"`
}

type metricInvitesGraph struct {
	AsOn    string `json:"as_on"`
	Credits int64  `json:"credits"`
	Debits  int64  `json:"debits"`
}

// InvitesMetricsResponse represents the invites metrics
type InvitesMetricsResponse struct {
	InvitesUnlocked                  int64                `json:"invites_unlocked"`
	InvitesUnlocked24Hours           int64                `json:"invites_unlocked_24h"`
	InvitesUnlocked7Days             int64                `json:"invites_unlocked_7d"`
	InvitesUsedPercentage            float64              `json:"invites_used_percentage"`
	InvitesGraph                     []metricInvitesGraph `json:"graph_invites"`
	UsersCompletedSignedUp           int64                `json:"users_completed_signed_up"`
	UsersCompletedOneArgument        int64                `json:"users_completed_one_argument"`
	UsersCompletedGivenOneAgree      int64                `json:"users_completed_given_one_agree"`
	UsersCompletedReceivedFiveAgrees int64                `json:"users_completed_received_five_agrees"`
	Users                            []metricUser         `json:"users"`
}

// HandleInvitesMetrics returns the metrics for the invites system
func (ta *TruAPI) HandleInvitesMetrics(w http.ResponseWriter, r *http.Request) {
	client := db.NewDBClient(ta.APIContext.Config)
	metrics := InvitesMetricsResponse{}
	var count int

	// Invites Unlocked
	_, err := client.Model(nil).QueryOne(pg.Scan(&count),
		`select sum(amount) from reward_ledger_entries where direction = ? and currency = ?;`,
		db.RewardLedgerEntryDirectionCredit,
		db.RewardLedgerEntryCurrencyInvite,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.InvitesUnlocked = int64(count)

	// Invites Unlocked 7 Days
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select sum(amount) from reward_ledger_entries where direction = ? and currency = ? and created_at > NOW() - interval '7 days';`,
		db.RewardLedgerEntryDirectionCredit,
		db.RewardLedgerEntryCurrencyInvite,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.InvitesUnlocked7Days = int64(count)

	// Invites Unlocked 24 Hours
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select sum(amount) from reward_ledger_entries where direction = ? and currency = ? and created_at > NOW() - interval '24 hours';`,
		db.RewardLedgerEntryDirectionCredit,
		db.RewardLedgerEntryCurrencyInvite,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.InvitesUnlocked24Hours = int64(count)

	// Invites Used Percentage
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select sum(amount) from reward_ledger_entries where direction = ? and currency = ?;`,
		db.RewardLedgerEntryDirectionDebit,
		db.RewardLedgerEntryCurrencyInvite,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.InvitesUsedPercentage = (float64(count) / float64(metrics.InvitesUnlocked)) * 100

	// Users Completed Signed Up
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
		fmt.Sprintf("[\"%s\"]", db.JourneyStepSignedUp),
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedSignedUp = int64(count)

	// Users Completed One Argument
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
		fmt.Sprintf("[\"%s\"]", db.JourneyStepOneArgument),
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedOneArgument = int64(count)

	// Users Completed Given One Agree
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
		fmt.Sprintf("[\"%s\"]", db.JourneyStepGivenOneAgree),
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedGivenOneAgree = int64(count)

	// Users Completed Receive Five Agrees
	_, err = client.Model(nil).QueryOne(pg.Scan(&count),
		`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
		fmt.Sprintf("[\"%s\"]", db.JourneyStepReceiveFiveAgrees),
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedReceivedFiveAgrees = int64(count)

	// Invites Graph
	_, err = client.Model(nil).Query(&metrics.InvitesGraph,
		`select 
			date(created_at) as as_on,
			sum(case direction when ? then amount else 0 end) as credits,
			sum(case direction when ? then amount else 0 end) as debits
		from 
			reward_ledger_entries where currency = ?
		group by 
			as_on
		order by 
			as_on;`,
		db.RewardLedgerEntryDirectionCredit,
		db.RewardLedgerEntryDirectionDebit,
		db.RewardLedgerEntryCurrencyInvite,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Users
	users := make([]db.User, 0)
	err = client.FindAll(&users)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, user := range users {
		mUser := metricUser{
			ID:               user.ID,
			FullName:         user.FullName,
			Username:         user.Username,
			Email:            user.Email,
			InvitesLeft:      user.InvitesLeft,
			JourneyCompleted: user.Meta.Journey,
		}

		if user.ReferredBy != 0 {
			referrer, err := client.UserByID(user.ReferredBy)
			if err != nil {
				render.Error(w, r, err.Error(), http.StatusInternalServerError)
				return
			}

			mUser.ReferrerUsername = referrer.Username
			mUser.ReferredEmail = referrer.Email
		}

		metrics.Users = append(metrics.Users, mUser)
	}

	render.Response(w, r, metrics, http.StatusOK)
}
