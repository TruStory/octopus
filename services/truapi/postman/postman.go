package postman

import (
	"html/template"

	
	"github.com/TruStory/octopus/services/truapi/context"
	
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
	To      []string
	CC      []string
	Subject string
	Body    string
}

// NewVanillaPostman creates the client without the truapi dependency
func NewVanillaPostman(region, sender, key, secret string) (*Postman, error) {
	// setting up all message templates
	box := packr.New("Email Templates", "./templates")
	templates := []string{
		"register", "invitation", "password-reset", "email-confirmation",
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
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
	})
	if err != nil {
		return nil, err
	}

	// returning the client
	return &Postman{
		Region:   region,
		Sender:   sender,
		CharSet:  "UTF-8",
		SES:      ses.New(session),
		Messages: messages,
	}, nil
}

// NewPostman creates the client to deliver SES emails
func NewPostman(config context.Config) (*Postman, error) {
	return NewVanillaPostman(config.AWS.Region, config.AWS.Sender, config.AWS.AccessKey, config.AWS.AccessSecret)
}

// Deliver sends the email to the designated recipient
func (postman *Postman) Deliver(message Message) error {
	cc, to := []*string{}, []*string{}
	for _, address := range message.CC {
		cc = append(cc, aws.String(address))
	}
	for _, address := range message.To {
		to = append(to, aws.String(address))
	}
	// Assemble the email.
	input := &ses.SendEmailInput{
		Source: aws.String(postman.Sender),
		Destination: &ses.Destination{
			CcAddresses: cc,
			ToAddresses: to,
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Charset: aws.String(postman.CharSet),
				Data:    aws.String(message.Subject),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(postman.CharSet),
					Data:    aws.String(message.Body),
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
