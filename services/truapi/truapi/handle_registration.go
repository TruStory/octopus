package truapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/TruStory/octopus/services/truapi/chttp"
	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/btcsuite/btcd/btcec"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

// RegistrationRequest is a JSON request body representing a twitter profile that a user wishes to register
type RegistrationRequest struct {
	AuthToken       string `json:"auth_token"`
	AuthTokenSecret string `json:"auth_token_secret"`
}

// RegistrationResponse is a JSON response body representing the result of registering a key
type RegistrationResponse struct {
	UserID               string                             `json:"userId"`
	Username             string                             `json:"username"` // deprecated. Use RegistrationTwitterProfileResponse.Username
	Fullname             string                             `json:"fullname"` // deprecated. Use RegistrationTwitterProfileResponse.Fullname
	Address              string                             `json:"address"`
	AuthenticationCookie string                             `json:"authenticationCookie"`
	TwitterProfile       RegistrationTwitterProfileResponse `json:"twitterProfile"`
}

// RegistrationTwitterProfileResponse is a JSON response body representing the TwitterProfile of a user
type RegistrationTwitterProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURI string `json:"avatarURI"`
}

// HandleRegistration takes a `RegistrationRequest` and returns a `RegistrationResponse`
func (ta *TruAPI) HandleRegistration(r *http.Request) chttp.Response {
	rr := new(RegistrationRequest)
	reqBytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	err = json.Unmarshal(reqBytes, &rr)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	// Get the Twitter User from the auth token
	twitterUser, err := getTwitterUser(ta.APIContext, rr.AuthToken, rr.AuthTokenSecret)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	return RegisterTwitterUser(ta, twitterUser)
}

// RegisterTwitterUser registers a new twitter user
func RegisterTwitterUser(ta *TruAPI, twitterUser *twitter.User) chttp.Response {
	user, err := CalibrateUser(ta, twitterUser)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	cookieValue, err := cookies.MakeLoginCookieValue(ta.APIContext, user)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	responseBytes, _ := json.Marshal(RegistrationResponse{
		UserID:               twitterUser.IDStr,
		Username:             twitterUser.ScreenName,
		Fullname:             twitterUser.Name,
		Address:              user.Address,
		AuthenticationCookie: cookieValue,
		TwitterProfile: RegistrationTwitterProfileResponse{
			Username:  user.Username,
			FullName:  user.FirstName,
			AvatarURI: user.AvatarURL,
		},
	})

	return chttp.SimpleResponse(201, responseBytes)
}

// CalibrateUser takes a twitter authenticated user and makes sure it has
// been properly calibrated in the database with all proper keypairs
func CalibrateUser(ta *TruAPI, twitterUser *twitter.User) (*db.User, error) {
	connectedAccount, err := ta.DBClient.ConnectedAccountByTypeAndID("twitter", fmt.Sprintf("%d", twitterUser.ID))
	if err != nil {
		return nil, err
	}

	if connectedAccount == nil {
		// this user is logging in for the first time, thus, register them
		connectedAccount = &db.ConnectedAccount{
			AccountType: "twitter",
			AccountID:   fmt.Sprintf("%d", twitterUser.ID),
			Meta: db.ConnectedAccountMeta{
				Email:     twitterUser.Email,
				Bio:       twitterUser.Description,
				Username:  twitterUser.ScreenName,
				FullName:  twitterUser.Name,
				AvatarURL: strings.Replace(twitterUser.ProfileImageURL, "_normal", "_bigger", 1),
			},
		}

		user, err := ta.DBClient.AddUserViaConnectedAccount(connectedAccount)
		if err != nil {
			return nil, err
		}

		// generating a signing keypair for the user,
		// if they don't have it yet
		if user.Address == "" {
			keyPair, err := makeNewKeyPair()
			if err != nil {
				return nil, err
			}
			keyPair.UserID = user.ID

			// registering on the chain
			pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
			if err != nil {
				return nil, err
			}
			address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
			if err != nil {
				return nil, err
			}

			// adding the keypair in the database
			err = ta.DBClient.Add(keyPair)
			if err != nil {
				return nil, err
			}
			// adding the address to the user
			err = ta.DBClient.AddAddressToUser(user.ID, address.String())
			if err != nil {
				return nil, err
			}
		}
	} else {
		// this user is already our user, so, we'll just update their meta fields to stay updated
		connectedAccount.Meta = db.ConnectedAccountMeta{
			Email:     twitterUser.Email,
			Bio:       twitterUser.Description,
			Username:  twitterUser.ScreenName,
			FullName:  twitterUser.Name,
			AvatarURL: strings.Replace(twitterUser.ProfileImageURL, "_normal", "_bigger", 1),
		}
		err = ta.DBClient.UpsertConnectedAccount(connectedAccount)
		if err != nil {
			return nil, err
		}
	}

	// finally fetching a fresh copy of the user and returning back
	user, err := ta.DBClient.UserByConnectedAccountTypeAndID(connectedAccount.AccountType, connectedAccount.AccountID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func makeNewKeyPair() (*db.KeyPair, error) {
	newKeyPair, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}

	// We are converting the private key of the new key pair in hex string,
	// then back to byte slice, and finally regenerating the private (suppressed) and public key from it.
	// This way, it returns the kind of public key that cosmos understands.
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), []byte(fmt.Sprintf("%x", newKeyPair.Serialize())))

	return &db.KeyPair{
		PrivateKey: fmt.Sprintf("%x", newKeyPair.Serialize()),
		PublicKey:  fmt.Sprintf("%x", pubKey.SerializeCompressed()),
	}, nil
}

func getTwitterUser(apiCtx truCtx.TruAPIContext, authToken string, authTokenSecret string) (*twitter.User, error) {
	ctx := context.Background()
	config := oauth1.NewConfig(apiCtx.Config.Twitter.APIKey, apiCtx.Config.Twitter.APISecret)

	httpClient := config.Client(ctx, oauth1.NewToken(authToken, authTokenSecret))
	twitterClient := twitter.NewClient(httpClient)
	accountVerifyParams := &twitter.AccountVerifyParams{
		IncludeEntities: twitter.Bool(false),
		SkipStatus:      twitter.Bool(true),
		IncludeEmail:    twitter.Bool(true),
	}
	user, _, err := twitterClient.Accounts.VerifyCredentials(accountVerifyParams)
	if err != nil {
		return nil, err
	}

	return user, nil
}
