package messages

import (
	"bytes"
	"fmt"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// MakeSignupMessage makes a new signup message
func MakeSignupMessage(client *postman.Postman, config context.Config, user db.User) (*postman.Message, error) {
	vars := struct {
		Name       string
		SignupLink string
	}{
		Name:       user.FullName,
		SignupLink: makeSignupLink(config, user),
	}

	var body bytes.Buffer
	if err := client.Messages["signup"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:       user.Email,
		Subject:  "Welcome to TruStory - you've been approved to join.",
		HTMLBody: body.String(),
		TextBody: body.String(),
	}, nil
}

func makeSignupLink(config context.Config, user db.User) string {
	return fmt.Sprintf("%s/signup?id=%d&token=%s", config.App.URL, user.ID, user.Token)
}
