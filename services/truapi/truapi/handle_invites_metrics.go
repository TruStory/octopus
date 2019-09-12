package truapi

import (
	"fmt"
	"net/http"

	"github.com/go-pg/pg"

	"github.com/TruStory/octopus/services/truapi/truapi/render"

	"github.com/TruStory/octopus/services/truapi/db"
)

// InvitesMetricsResponse represents the invites metrics
type InvitesMetricsResponse struct {
	InvitesUnlocked           int64     `json:"invites_unlocked"`
	InvitesUnlocked24Hours    int64     `json:"invites_unlocked_24h"`
	InvitesUnlocked7Days      int64     `json:"invites_unlocked_7d"`
	InvitesUsedPercentage     float64   `json:"invites_used_percentage"`
	UsersCompletedSignedUp    int64     `json:"users_completed_signed_up"`
	UsersCompletedOneArgument int64     `json:"users_completed_one_argument"`
	UsersCompletedFiveAgrees  int64     `json:"users_completed_five_agrees"`
	Users                     []db.User `json:"users"`
}

// HandleInvitesMetrics returns the metrics for the invites system
func (ta *TruAPI) HandleInvitesMetrics(w http.ResponseWriter, r *http.Request) {
	client := db.NewDBClient(ta.APIContext.Config)
	metrics := InvitesMetricsResponse{}
	var count int

	// Invites Unlocked
	_, err := client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
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
	_, err = client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
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
	_, err = client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
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
	_, err = client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
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
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
			fmt.Sprintf("[\"%s\"]", db.JourneyStepSignedUp),
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedSignedUp = int64(count)

	// Users Completed One Argument
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
			fmt.Sprintf("[\"%s\"]", db.JourneyStepOneArgument),
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedOneArgument = int64(count)

	// Users Completed Five Agrees
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users where meta->'journey' @> ? and deleted_at is null;`,
			fmt.Sprintf("[\"%s\"]", db.JourneyStepFiveAgrees),
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersCompletedFiveAgrees = int64(count)

	// Users
	users := make([]db.User, 0)
	err = client.FindAll(&users)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.Users = users

	render.Response(w, r, metrics, http.StatusOK)
}
