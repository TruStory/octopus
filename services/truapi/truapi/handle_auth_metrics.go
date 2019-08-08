package truapi

import (
	"net/http"

	"github.com/go-pg/pg"

	"github.com/TruStory/octopus/services/truapi/truapi/render"

	"github.com/TruStory/octopus/services/truapi/db"
)

// AuthMetricsResponse represents the auth metrics
type AuthMetricsResponse struct {
	TotalUsers                  int64   `json:"total_users"`
	TotalUsers7Days             int64   `json:"total_users_7d"`
	TotalUsers24Hours           int64   `json:"total_users_24h"`
	UsersViaTwitter             int64   `json:"users_via_twitter"`
	UsersViaEmail               int64   `json:"users_via_email"`
	UsersViaTwitter7Days        int64   `json:"users_via_twitter_7d"`
	UsersViaEmail7Days          int64   `json:"users_via_email_7d"`
	VerifiedEmailUserPercentage float64 `json:"verified_email_user_percentage"`
}

// HandleAuthMetrics returns the metrics for the auth flow
func (ta *TruAPI) HandleAuthMetrics(w http.ResponseWriter, r *http.Request) {
	client := db.NewDBClient(ta.APIContext.Config)
	metrics := AuthMetricsResponse{}
	var count int

	// Total Users
	_, err := client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
		`select count(*) from users;`,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.TotalUsers = int64(count)

	// Total Users 7 Days
	_, err = client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
		`select count(*) from users where created_at > NOW() - interval '7 days';`,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.TotalUsers7Days = int64(count)

	// Total Users 24 Hours
	_, err = client.Model((*db.User)(nil)).QueryOne(pg.Scan(&count),
		`select count(*) from users where created_at > NOW() - interval '24 hours';`,
	)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.TotalUsers24Hours = int64(count)

	// Users Via Twitter
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users left join connected_accounts on users.id = connected_accounts.user_id where account_type is not null;`,
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersViaTwitter = int64(count)

	// Users via Email
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users left join connected_accounts on users.id = connected_accounts.user_id where account_type is null;`,
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersViaEmail = int64(count)

	// Users Via Twitter 7 Days
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users left join connected_accounts on users.id = connected_accounts.user_id where account_type is not null and users.created_at > NOW() - interval '7 days';`,
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersViaTwitter7Days = int64(count)

	// Users via Email
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users left join connected_accounts on users.id = connected_accounts.user_id where account_type is null and users.created_at > NOW() - interval '7 days';`,
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.UsersViaEmail7Days = int64(count)

	// Verified Email User Percentage
	_, err = client.Model((*db.User)(nil)).
		QueryOne(pg.Scan(&count),
			`select count(*) from users left join connected_accounts on users.id = connected_accounts.user_id where account_type is null and users.verified_at is not null;`,
		)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	metrics.VerifiedEmailUserPercentage = (float64(count) / float64(metrics.UsersViaEmail)) * 100

	render.Response(w, r, metrics, http.StatusOK)
}
