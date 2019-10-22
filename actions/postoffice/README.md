## Postoffice

This module allows mass campaigns to be sent via truapi's Postman. Think of this as the backend office where all the mailers/campaigns are stored, and then handed off to Postman for the delivery.

### How to make campaign
Each campaign must adhere to the `Campaign` interface:

```go
import (
	"github.com/TruStory/octopus/services/truapi/postman"
)

type Campaign interface {
	GetRecipients(dbClient *db.Client) Recipients
	GetMessage(client *postman.Postman, recipient Recipient) (*postman.Message, error)
	RunPostProcess(dbClient *db.Client, recipient Recipient) error
}
```

The `Recipient` struct looks like:

```go
type Recipient struct {
	Email string
	User  *db.User
}

type Recipients []Recipient
```

### How to register a campaign
Once a new campaign is created, it must be added to the `registry` in the `main.go` file. Only the added campaigns in the registry are available to the CLI.

```go
registry["waitlist-approval"] = (*campaigns.WaitlistApprovalCampaign)(nil)
```

### How to trigger a campaign
To trigger a campaign, one must run the following command from the CLI:

```
go run *.go CAMAPAIGN_NAME_REGISTERED_IN_REGISTRY EMAIL_ADDRESS_OF_SENDER

go run *.go waitlist-approval preethi@trustory.io
```

If the ENV variables are not set on the environment, they can be passed in via CLI like following:

```
AWS_ACCESS_KEY=XXX AWS_ACCESS_SECRET=XXX go run *.go waitlist-approval preethi@trustory.io
```