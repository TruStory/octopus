package postman

import (
	"html/template"

	"github.com/TruStory/octopus/services/truapi/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	packr "github.com/gobuffalo/packr/v2"
)

// Postman is the client
type Postman struct {
	Region   string
	Sender   string
	CharSet  string
	SES      *ses.SES
	Messages map[string]*template.Template
}

// Message represents an email that can be sent
type Message struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

// NewPostman creates the client to deliver SES emails
func NewPostman(config context.Config) (*Postman, error) {
	// setting up all message templates
	box := packr.New("Email Templates", "./templates")
	templates := []string{
		"signup",
	}
	messages := make(map[string]*template.Template)
	for _, templateName := range templates {
		templateFilename := templateName + ".html.tmpl"
		filename, err := box.FindString(templateFilename)
		if err != nil {
			return nil, err
		}
		parsedTemplate, err := template.New(templateFilename).Parse(filename)
		if err != nil {
			return nil, err
		}

		messages[templateName] = parsedTemplate
	}
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWS.Region)},
	)
	if err != nil {
		return nil, err
	}

	// returning the client
	return &Postman{
		Region:   config.AWS.Region,
		Sender:   config.AWS.Sender,
		CharSet:  "UTF-8",
		SES:      ses.New(session),
		Messages: messages,
	}, nil
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
