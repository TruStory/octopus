package messages

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// MakeRegisterMessage makes a new register message
func MakeRegisterMessage(client *postman.Postman, config context.Config, user db.User) (*postman.Message, error) {
	vars := struct {
		Name         string
		RegisterLink string
	}{
		Name:         user.FullName,
		RegisterLink: makeRegisterLink(config, user),
	}

	var body bytes.Buffer
	if err := client.Messages["register"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:      []string{user.Email},
		Subject: "Getting you started with TruStory Beta",
		Body:    string(blackfriday.Run(body.Bytes())),
	}, nil
}

func makeRegisterLink(config context.Config, user db.User) string {
	return fmt.Sprintf("%s/register?id=%d&token=%s", config.App.URL, user.ID, user.Token)
}
