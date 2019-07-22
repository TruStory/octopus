package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// UpdateClaimImage represents the JSON request for setting a claim image
type UpdateClaimImage struct {
	ClaimID uint64 `json:"claim_id"`
	URL     string `json:"claim_image_url"`
}

// HandleClaimImage handles requests for claim image
func (ta *TruAPI) HandleClaimImage(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPut:
		return ta.updateClaimImage(r)
	default:
		return chttp.SimpleErrorResponse(404, Err404ResourceNotFound)
	}
}

func (ta *TruAPI) updateClaimImage(r *http.Request) chttp.Response {
	request := &UpdateClaimImage{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	user, ok := r.Context().Value(userContextKey).(*cookies.AuthenticatedUser)
	if !ok || user == nil {
		return chttp.SimpleErrorResponse(401, Err401NotAuthenticated)
	}

	claim := ta.claimResolver(r.Context(), queryByClaimID{ID: request.ClaimID})
	settings := ta.settingsResolver(r.Context())

	// preethis cosmos address for admin privilieges
	if claim.Creator.String() != user.Address && !contains(settings.ClaimAdmins, user.Address) {
		return chttp.SimpleErrorResponse(403, Err403NotAuthorized)
	}

	claimImageURL := &db.ClaimImage{
		ClaimID:       request.ClaimID,
		ClaimImageURL: request.URL,
	}
	err = ta.DBClient.AddClaimImage(claimImageURL)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}
	return chttp.SimpleResponse(200, nil)
}
