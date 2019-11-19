package twilio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Deliverable struct {
	client  *Client
	message *Message
	to      string
}

func (d *Deliverable) Send() error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := d.getHTTPRequest()
	if err != nil {
		return err
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	var errResp map[string]interface{}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&errResp)
	if err != nil {
		return err
	}
	return fmt.Errorf("error sending: %s", errResp["message"])
}

func (d *Deliverable) getHTTPRequest() (*http.Request, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(d.client.endpoint+"/Accounts/%s/Messages.json", d.client.sid),
		d.toHTTPDataReader(),
	)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(d.client.sid, d.client.token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func (d *Deliverable) toHTTPDataReader() *strings.Reader {
	data := url.Values{}
	data.Set("To", d.to)
	data.Set("From", d.client.from)
	data.Set("Body", d.message.Get())
	return strings.NewReader(data.Encode())
}
