package truapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/btcec"

	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
)

// UserResponse is a JSON response body representing the result of User
type UserResponse struct {
	UserID         string                     `json:"userId"`
	Username       string                     `json:"username"` // deprecated. Use UserTwitterProfileResponse.Username
	Fullname       string                     `json:"fullname"` // deprecated. Use UserTwitterProfileResponse.Fullname
	Address        string                     `json:"address"`
	TwitterProfile UserTwitterProfileResponse `json:"twitterProfile"`
}

// UserTwitterProfileResponse is a JSON response body representing the TwitterProfile of a user
type UserTwitterProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURI string `json:"avatarURI"`
}

// UpdateUserViaTokenRequest updates a user via one-time use token
type UpdateUserViaTokenRequest struct {
	ID       uint64 `json:"id"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateUserViaCookieRequest updates a user's profile fields
type UpdateUserViaCookieRequest struct {
	// Profile fields
	Profile *db.UserProfile `json:"profile,omitempty"`

	// Password fields
	Password *db.UserPassword `json:"password,omitempty"`
}

// HandleUserDetails takes a `UserRequest` and returns a `UserResponse`
func (ta *TruAPI) HandleUserDetails(r *http.Request) chttp.Response {
	switch r.Method {
	case http.MethodPut:
		return ta.updateUserDetails(r)
	default:
		return ta.getUserDetails(r)
	}
}

func (ta *TruAPI) updateUserDetails(r *http.Request) chttp.Response {
	// There are two scenarios when a user can be updated:
	// a.) When users are setting their username + password (signing up) after admin approval (one-time process)
	// b.) When users are updating their profile (bio, photo, etc) from their account settings
	// To update a user, either the user cookie has to be present (for [scenario b]),
	// or a token has to be present (for [scenario a]).

	// attempt to get the cookie
	u, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	fmt.Println(err, u)
	if err == http.ErrNoCookie {
		// no cookie present; proceed via token
		return ta.updateUserDetailsViaRequestToken(r)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// cookie found, proceed via cookie
	return ta.updateUserDetailsViaCookie(r)
}

func (ta *TruAPI) updateUserDetailsViaRequestToken(r *http.Request) chttp.Response {
	var request UpdateUserViaTokenRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	err = ta.DBClient.SignupUser(request.ID, request.Token, request.Username, request.Password)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	// MILESTONE -- successfully signedup, let's give the user an address (registering user on the chain)
	newKeyPair, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}
	// We are converting the private key of the new key pair in hex string,
	// then back to byte slice, and finally regenerating the private (suppressed) and public key from it.
	// This way, it returns the kind of public key that cosmos understands.
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), []byte(fmt.Sprintf("%x", newKeyPair.Serialize())))
	keyPair := &db.KeyPair{
		TwitterProfileID: request.ID,
		PrivateKey:       fmt.Sprintf("%x", newKeyPair.Serialize()),
		PublicKey:        fmt.Sprintf("%x", pubKey.SerializeCompressed()),
	}
	pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}
	address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}
	err = ta.DBClient.AddAddressToUser(request.ID, address.String())
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)

}

func (ta *TruAPI) updateUserDetailsViaCookie(r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	var request UpdateUserViaCookieRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	// if user wants to change their password
	if request.Password != nil {
		err = ta.DBClient.UpdatePassword(user.ID, request.Password)
		if err != nil {
			return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
		}

		return chttp.SimpleResponse(http.StatusOK, nil)
	}

	// if user wants to change their profile
	if request.Profile != nil {
		err = ta.DBClient.UpdateProfile(user.ID, request.Profile)
		if err != nil {
			return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
		}

		return chttp.SimpleResponse(http.StatusOK, nil)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)
}

func (ta *TruAPI) getUserDetails(r *http.Request) chttp.Response {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		return chttp.SimpleErrorResponse(401, err)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	twitterProfile, err := ta.DBClient.TwitterProfileByID(user.TwitterProfileID)
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// Chain was restarted and DB was wiped so Address and TwitterProfileID contained in cookie is stale.
	if twitterProfile.ID == 0 {
		return chttp.SimpleErrorResponse(401, err)
	}

	responseBytes, _ := json.Marshal(UserResponse{
		UserID:   strconv.FormatInt(twitterProfile.ID, 10),
		Fullname: twitterProfile.FullName,
		Username: twitterProfile.Username,
		Address:  twitterProfile.Address,
		TwitterProfile: UserTwitterProfileResponse{
			Username:  twitterProfile.Username,
			FullName:  twitterProfile.FullName,
			AvatarURI: twitterProfile.AvatarURI,
		},
	})

	return chttp.SimpleResponse(200, responseBytes)
}
