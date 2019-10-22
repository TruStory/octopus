package truapi

import (
	"encoding/json"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UserExportRequest represents the JSON request for exporting the private key
type UserExportRequest struct {
	EncryptedPrivateKey *string `json:"encrypted_private_key"`
}

// HandleUserExport takes a `UserExportRequest` and returns a 200 response
func (ta *TruAPI) HandleUserExport(w http.ResponseWriter, r *http.Request) {
	// only support POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request := &UserExportRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	keyPair, err := ta.DBClient.KeyPairByUserID(user.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if keyPair.PrivateKey == "" {
		render.Error(w, r, "key already exported", http.StatusBadRequest)
		return
	}

	if request.EncryptedPrivateKey != nil {
		err = ta.DBClient.ReplacePrivateKeyWithEncryptedPrivateKey(user.ID, *request.EncryptedPrivateKey)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	render.Response(w, r, keyPair, http.StatusOK)
}
