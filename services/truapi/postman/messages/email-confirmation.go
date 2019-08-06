package messages

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// MakeEmailConfirmationMessage makes a new email confirmation message
func MakeEmailConfirmationMessage(client *postman.Postman, config context.Config, user db.User) (*postman.Message, error) {
	vars := struct {
		FullName         string
		VerificationLink string
	}{
		FullName:         user.FullName,
		VerificationLink: makeVerificationLink(config, user),
	}

	var body bytes.Buffer
	if err := client.Messages["email-confirmation"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:      []string{user.Email},
		Subject: "Confirm your email address",
		Body:    string(blackfriday.Run(body.Bytes())),
	}, nil
}

func makeVerificationLink(config context.Config, user db.User) string {
	return fmt.Sprintf("%s/verify?id=%d&token=%s", config.App.URL, user.ID, user.Token)
}
