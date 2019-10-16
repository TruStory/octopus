package campaigns

import (
	"strconv"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
	"github.com/TruStory/octopus/services/truapi/postman/messages"
)

var _ Campaign = (*VerificationAttemptCampaign)(nil)

// VerificationAttemptCampaign is the campaign to attempt verification of the new users
type VerificationAttemptCampaign struct{}

// GetRecipients returns all the recipients of the campaign
func (campaign *VerificationAttemptCampaign) GetRecipients(dbClient *db.Client) (Recipients, error) {
	users, err := dbClient.UnverifiedNewUsers()
	if err != nil {
		return Recipients{}, err
	}

	limit, err := strconv.Atoi(getEnv("VERIFICATION_ATTEMPT_LIMIT", "3"))
	if err != nil {
		return Recipients{}, err
	}
	delay, err := strconv.Atoi(getEnv("VERIFICATION_ATTEMPT_DELAY", "48"))
	if err != nil {
		return Recipients{}, err
	}

	recipients := Recipients{}
	for _, user := range users {
		if user.VerificationAttemptCount < limit && isAttemptDelayedBy(user.LastVerificationAttemptAt, delay) {
			recipients = append(recipients, Recipient{
				Email: user.Email,
				User:  &user,
			})
		}

	}

	return recipients, nil
}

// GetMessage returns a message that is to be sent to a particular recipient
func (campaign *VerificationAttemptCampaign) GetMessage(client *postman.Postman, recipient Recipient) (*postman.Message, error) {
	config := truCtx.Config{
		App: truCtx.AppConfig{
			URL: mustEnv("APP_URL"),
		},
	}

	message, err := messages.MakeEmailConfirmationMessage(client, config, *recipient.User)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func isAttemptDelayedBy(lastAt time.Time, hours int) bool {
	if lastAt.IsZero() {
		return true // if no previous attempt is made, make one now
	}

	return lastAt.
		Add(time.Duration(hours) * time.Hour). // if the time after adding the delay,
		Before(time.Now())                     // is still in the past
}

func (campaign *VerificationAttemptCampaign) RunPostProcess(dbClient *db.Client, recipient Recipient) error {
	return dbClient.RecordVerificationAttempt(recipient.User.ID)
}
