package campaigns

import (
	"bytes"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman"
	"github.com/russross/blackfriday/v2"
)

var recipients = Recipients{
	Recipient{Email: "mohit.mamoria@gmail.com"},
	Recipient{Email: "mamoria.mohit@gmail.com"},
}

var _ Campaign = (*WaitlistApprovalCampaign)(nil)

// WaitlistApprovalCampaign is the campaign to approve all the waitlist users
type WaitlistApprovalCampaign struct{}

// GetRecipients returns all the recipients of the campaign
func (campaign *WaitlistApprovalCampaign) GetRecipients(_ *db.Client) (Recipients, error) {
	return recipients, nil
}

// GetMessage returns a message that is to be sent to a particular recipient
func (campaign *WaitlistApprovalCampaign) GetMessage(client *postman.Postman, recipient Recipient) (*postman.Message, error) {
	vars := struct {
		RegisterLink string
	}{
		RegisterLink: "https://beta.trustory.io/register",
	}

	var body bytes.Buffer
	if err := client.Messages["register"].Execute(&body, vars); err != nil {
		return nil, err
	}

	return &postman.Message{
		To:      []string{recipient.Email},
		Subject: "Getting you started with TruStory Beta",
		Body:    string(blackfriday.Run(body.Bytes())),
	}, nil
}

func (campaign *WaitlistApprovalCampaign) RunPostProcess(_ *db.Client, _ Recipient) error {
	return nil
}
