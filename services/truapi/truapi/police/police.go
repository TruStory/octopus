package police

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Officer struct {
	accountsid string
	token      string
	verifysid  string
	endpoint   string
}

func NewOfficer(accountsid, token, verifysid string) *Officer {
	return &Officer{
		accountsid: accountsid,
		token:      token,
		verifysid:  verifysid,
		endpoint:   "https://verify.twilio.com/v2/",
	}
}

func (o *Officer) Initiate(to, channel string) error {
	req, err := o.makeInitiateHTTPRequest(to, channel)
	if err != nil {
		return err
	}

	response, err := getHTTPClient().Do(req)
	if err != nil {
		return err
	}

	return returnErrIfAny(response)
}

func (o *Officer) Check(to, code string) error {
	req, err := o.makeCheckHTTPRequest(to, code)
	if err != nil {
		return err
	}

	response, err := getHTTPClient().Do(req)
	if err != nil {
		return err
	}

	return returnErrIfAny(response)
}

func (o *Officer) makeInitiateHTTPRequest(to, channel string) (*http.Request, error) {
	return o.makeHTTPRequest(
		"POST",
		fmt.Sprintf(o.endpoint+"/Services/%s/Verifications", o.verifysid),
		makeInitiateHTTPBody(to, channel),
	)
}

func (o *Officer) makeCheckHTTPRequest(to, code string) (*http.Request, error) {
	return o.makeHTTPRequest(
		"POST",
		fmt.Sprintf(o.endpoint+"/Services/%s/VerificationCheck", o.verifysid),
		makeCheckHTTPBody(to, code),
	)
}

func (o *Officer) makeHTTPRequest(method, endpoint string, data *strings.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, endpoint, data)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(o.accountsid, o.token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func makeInitiateHTTPBody(to, channel string) *strings.Reader {
	data := url.Values{}
	data.Set("To", to)
	data.Set("Channel", channel)
	return strings.NewReader(data.Encode())
}

func makeCheckHTTPBody(to, code string) *strings.Reader {
	data := url.Values{}
	data.Set("To", to)
	data.Set("Code", code)
	return strings.NewReader(data.Encode())
}

func getHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

func returnErrIfAny(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	var errResp map[string]interface{}
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&errResp)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s", errResp["message"])
}
