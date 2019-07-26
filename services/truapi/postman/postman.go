package postman

import (
	"github.com/TruStory/octopus/services/truapi/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

// Postman is the client
type Postman struct {
	Region  string
	Sender  string
	CharSet string
	SES     *ses.SES
}

// Message represents an email that can be sent
type Message struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

// NewPostman creates the client to deliver SES emails
func NewPostman(config context.Config) *Postman {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWS.Region)},
	)
	if err != nil {
		panic(err)
	}
	return &Postman{
		Region:  config.AWS.Region,
		Sender:  config.AWS.Sender,
		CharSet: "UTF-8",
		SES:     ses.New(session),
	}
}

// Deliver sends the email to the designated recipient
func (postman *Postman) Deliver(message Message) error {
	// Assemble the email.
	input := &ses.SendEmailInput{
		Source: aws.String(postman.Sender),
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(message.To),
			},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Charset: aws.String(postman.CharSet),
				Data:    aws.String(message.Subject),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(postman.CharSet),
					Data:    aws.String(message.HTMLBody),
				},
				Text: &ses.Content{
					Charset: aws.String(postman.CharSet),
					Data:    aws.String(message.TextBody),
				},
			},
		},
	}

	// Attempt to send the email.
	_, err := postman.SES.SendEmail(input)

	if err != nil {
		return err
	}

	return nil
}
