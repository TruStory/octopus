package truapi

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
)

// EmailRegistrationRequest represents the request to have a user signup via email
type EmailRegistrationRequest struct {
	WaitlistID    uint64 `json:"waitlist_id"`
	WaitlistToken string `json:"waitlist_token"`
}

// HandleRegistrationViaEmail takes a `RegistrationRequest` and returns a `RegistrationResponse`
func (ta *TruAPI) HandleRegistrationViaEmail(r *http.Request) chttp.Response {
	var req EmailRegistrationRequest
	reqBytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	err = json.Unmarshal(reqBytes, &req)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	return chttp.SimpleResponse(200, reqBytes)
}
