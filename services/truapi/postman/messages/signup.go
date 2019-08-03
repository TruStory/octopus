package messages

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"

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

	compiledBody := string(blackfriday.Run(body.Bytes()))
	return &postman.Message{
		To:       user.Email,
		Subject:  "Getting you started with TruStory Beta",
		HTMLBody: compiledBody,
		TextBody: compiledBody,
	}, nil
}

func makeSignupLink(config context.Config, user db.User) string {
	return fmt.Sprintf("%s/signup?id=%d&token=%s", config.App.URL, user.ID, user.Token)
}
