package twilio

type Client struct {
	sid      string
	token    string
	from     string
	endpoint string
}

func NewClient(sid, token, from string) *Client {
	return &Client{
		sid:      sid,
		token:    token,
		from:     from,
		endpoint: "https://api.twilio.com/2010-04-01",
	}
}

func (c *Client) Send(to string, message *Message) error {
	deliverable := &Deliverable{
		client:  c,
		to:      to,
		message: message,
	}

	err := deliverable.Send()
	if err != nil {
		return err
	}

	return nil
}
