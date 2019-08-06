package messages

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// MakeInvitationMessage makes a new invitation message
func MakeInvitationMessage(client *postman.Postman, config context.Config, name, email string, referrer db.User) (*postman.Message, error) {
	vars := struct {
		FullName   string
		Referrer   db.User
		SignupLink string
	}{
		FullName:   name,
		Referrer:   referrer,
		SignupLink: makeReferralSignupLink(config, referrer),
	}

	var body bytes.Buffer
	if err := client.Messages["invitation"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:      []string{email},
		CC:      []string{referrer.Email},
		Subject: fmt.Sprintf("You’ve been invited to TruStory Beta by %s!", referrer.FullName),
		Body:    string(blackfriday.Run(body.Bytes())),
	}, nil
}

func makeReferralSignupLink(config context.Config, referrer db.User) string {
	return fmt.Sprintf("%s/signup?referrer=%s", config.App.URL, referrer.Address)
}
