package truapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UserExportRequest represents the JSON request for exporting the private key
type UserExportRequest struct {
	EncryptedPrivateKey *string `json:"encrypted_private_key"`
}

// UserSetKeyRequest is the request to set key on a user
type UserSetKeyRequest struct {
	PrivateKey string `json:"private_key"` // it is an encrypted version
	PublicKey  string `json:"public_key"`  // we'll use it to calculate the address on the server
}

// HandleUserSetKey takes a `UserSetKeyRequest` and returns a 200 response
func (ta *TruAPI) HandleUserSetKey(w http.ResponseWriter, r *http.Request) {
	// only support POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request := &UserSetKeyRequest{}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = validateUserSetKeyRequest(*request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	returnableUser, err := ta.DBClient.UserByID(user.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	if returnableUser.Address != "" {
		render.Error(w, r, "key already set", http.StatusBadRequest)
		return
	}

	// adding the key pair
	keyPair := makeKeyPairFromRequest(*request)
	address, err := ta.registerUserOnChain(user.ID, keyPair)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	returnableUser.Address = address.String()
	userResponse, err := ta.createUserResponse(r.Context(), returnableUser, false)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Response(w, r, userResponse, http.StatusOK)
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

func validateUserSetKeyRequest(request UserSetKeyRequest) error {
	request.PrivateKey = strings.TrimSpace(request.PrivateKey)
	request.PublicKey = strings.TrimSpace(request.PublicKey)

	if request.PrivateKey == "" || request.PublicKey == "" {
		return errors.New("key pair cannot be empty")
	}

	err := validateKeyPairRequest(request)
	if err != nil {
		return err
	}

	return nil
}

func validateKeyPairRequest(request UserSetKeyRequest) error {
	pk, err := newPubKey(request.PublicKey)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromHex(pk.Address().String())
	if err != nil {
		return err
	}

	return nil
}

func newPubKey(pk string) (res crypto.PubKey, err error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return
	}
	var pkSecp secp256k1.PubKeySecp256k1
	copy(pkSecp[:], pkBytes[:])
	return pkSecp, nil
}

func (ta *TruAPI) registerUserOnChain(userID int64, keyPair *db.KeyPair) (sdk.AccAddress, error) {
	// registering the keypair
	pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
	if err != nil {
		return nil, err
		// render.Error(w, r, err.Error(), http.StatusInternalServerError)
		// return
	}
	address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
	if err != nil {
		return nil, err
	}

	// adding the keypair in the database
	keyPair.UserID = userID
	err = ta.DBClient.Add(keyPair)
	if err != nil {
		return nil, err
	}
	err = ta.DBClient.AddAddressToUser(userID, address.String())
	if err != nil {
		return nil, err
	}

	return address, nil
}

func makeKeyPairFromRequest(keyPair UserSetKeyRequest) *db.KeyPair {
	return &db.KeyPair{
		PrivateKey:          "",
		PublicKey:           keyPair.PublicKey,
		EncryptedPrivateKey: keyPair.PrivateKey,
	}
}
