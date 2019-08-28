package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// AddQuestionRequest represents the JSON request for adding a question
type AddQuestionRequest struct {
	ClaimID    int64  `json:"claim_id,omitempty"`
	Body       string `json:"body"`
}

type DeleteQuestionRequest struct {
	ID    int64  `json:"id,omitempty"`
}

// HandleQuestion handles requests for questions
func (ta *TruAPI) HandleQuestion(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPost:
		return ta.handleCreateQuestion(r)
	case http.MethodPut:
		return ta.handleDeleteQuestion(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) handleCreateQuestion(r *http.Request) chttp.Response {
	request := &AddQuestionRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	question := &db.Question{
		ClaimID:    request.ClaimID,
		Body:       request.Body,
		Creator:    user.Address,
	}
	err = ta.DBClient.AddQuestion(question)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	respBytes, err := json.Marshal(question)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}

	return chttp.SimpleResponse(200, respBytes)
}



func (ta *TruAPI) handleDeleteQuestion(r *http.Request) chttp.Response {
	request := &DeleteQuestionRequest{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	settings := ta.settingsResolver(r.Context())
	if !contains(settings.ClaimAdmins, user.Address) {
		return chttp.SimpleErrorResponse(403, Err403NotAuthorized)
	}

	question, err := ta.DBClient.QuestionByID(request.ID)
	if err != nil || question == nil {
		return chttp.SimpleErrorResponse(500, err)
	}

	err = ta.DBClient.DeleteQuestion(request.ID)

	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}

	return chttp.SimpleResponse(200, nil)
}

