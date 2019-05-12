package truapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"

	"github.com/TruStory/octopus/services/truapi/chttp"
)

// FlagStoryRequest represents the JSON request for flagging a story
type FlagStoryRequest struct {
	StoryID int64 `json:"story_id"`
}

// HandleFlagStory takes a `FlagStoryRequest` and returns a 200 response
func (ta *TruAPI) HandleFlagStory(r *http.Request) chttp.Response {
	request := &FlagStoryRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user := r.Context().Value(userContextKey)
	if user == nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// add data to table
	flaggedStory := &db.FlaggedStory{
		StoryID:   request.StoryID,
		Creator:   user.(*cookies.AuthenticatedUser).Address,
		CreatedOn: time.Now(),
	}
	err = ta.DBClient.UpsertFlaggedStory(flaggedStory)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	return chttp.SimpleResponse(200, nil)
}
