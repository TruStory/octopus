package campaigns

import (
	"fmt"
	"os"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// Recipient is anyone who can receive an email
type Recipient struct {
	Email string
	User  *db.User // this field is filled if the email is going to a TruStory user
}

// Recipients is the slice of Recipient
type Recipients []Recipient

// Campaign is the interface that every mass mailer campaign has to implement
type Campaign interface {
	GetRecipients(dbClient *db.Client) (Recipients, error)
	GetMessage(client *postman.Postman, recipient Recipient) (*postman.Message, error)
	RunPostProcess(dbClient *db.Client, recipient Recipient) error
}

func mustEnv(env string) string {
	val := os.Getenv(env)
	if val == "" {
		panic(fmt.Sprintf("must provide %s variable", env))
	}
	return val
}

func getEnv(env, defaultValue string) string {
	val := os.Getenv(env)
	if val != "" {
		return val
	}
	return defaultValue
}
