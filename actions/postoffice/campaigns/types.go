package campaigns

import (
	"github.com/TruStory/octopus/services/truapi/postman"
)

// Recipient is anyone who can receive an email
type Recipient struct {
	Email string
}

// Recipients is the slice of Recipient
type Recipients []Recipient

// Campaign is the interface that every mass mailer campaign has to implement
type Campaign interface {
	GetRecipients() Recipients
	GetMessage(client *postman.Postman, recipient Recipient) (*postman.Message, error)
}
