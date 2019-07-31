package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
)

// PingRequest is an empty JSON request
type PingRequest struct{}

// PingResponse is a JSON response body representing the result of Ping
type PingResponse struct {
	Pong bool `json:"pong"`
}

// HandlePing takes a `PingRequest` and returns a `PingResponse`
func (ta *TruAPI) HandlePing(r *http.Request) chttp.Response {
	ca, _ := ta.DBClient.ConnectedAccountByTypeAndID("twitter", "20765528")
	ca.Meta.Bio = "updated bio"
	err := ta.DBClient.UpsertConnectedAccount(ca)
	if err != nil {
		panic(err)
	}
	responseBytes, _ := json.Marshal(PingResponse{
		Pong: true,
	})

	return chttp.SimpleResponse(200, responseBytes)
}
