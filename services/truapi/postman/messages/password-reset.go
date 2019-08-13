package messages

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
)

// MakePasswordResetMessage makes a new password reset message
func MakePasswordResetMessage(client *postman.Postman, config context.Config, user db.User, prt db.PasswordResetToken) (*postman.Message, error) {
	vars := struct {
		Username  string
		ResetLink string
	}{
		Username:  user.Username,
		ResetLink: makeResetLink(config, user, prt),
	}

	var body bytes.Buffer
	if err := client.Messages["password-reset"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:      []string{user.Email},
		Subject: "Reset your password?",
		Body:    string(blackfriday.Run(body.Bytes())),
	}, nil
}

func makeResetLink(config context.Config, user db.User, prt db.PasswordResetToken) string {
	url := joinPath(config.App.URL, "/recovery/reset")
	return fmt.Sprintf("%s?id=%d&token=%s", url, user.ID, prt.Token)
}
