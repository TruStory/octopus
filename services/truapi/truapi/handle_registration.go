package truapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
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
	UserID      string                          `json:"userId"`
	Address     string                          `json:"address"`
	UserMeta    db.UserMeta                     `json:"userMeta"`
	UserProfile RegistrationUserProfileResponse `json:"userProfile"`

	// deprecated
	TwitterProfile RegistrationTwitterProfileResponse `json:"twitterProfile"`
}

// RegistrationTwitterProfileResponse is a JSON response body representing the TwitterProfile of a user
// deprecated: use RegistrationUserProfileResponse instead
type RegistrationTwitterProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURI string `json:"avatarURI"`
}

// RegistrationUserProfileResponse is a JSON body representing profile info of a user
type RegistrationUserProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
}

// HandleRegistration takes a `RegistrationRequest` and returns a `RegistrationResponse`
func (ta *TruAPI) HandleRegistration(w http.ResponseWriter, r *http.Request) {
	rr := new(RegistrationRequest)
	reqBytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(reqBytes, &rr)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the Twitter User from the auth token
	twitterUser, err := getTwitterUser(ta.APIContext, rr.AuthToken, rr.AuthTokenSecret)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, new, err := RegisterTwitterUser(ta, twitterUser)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	cookie, err := cookies.GetLoginCookie(ta.APIContext, user)
	if err != nil {
		render.LoginError(w, r, ErrServerError, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie)
	response, err := ta.createUserResponse(user, new)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Response(w, r, response, http.StatusOK)
}

// RegisterTwitterUser registers a new twitter user
func RegisterTwitterUser(ta *TruAPI, twitterUser *twitter.User) (*db.User, bool, error) {
	user, new, err := CalibrateUser(ta, twitterUser, "")
	if err != nil {
		return nil, false, err
	}

	err = ta.DBClient.TouchLastAuthenticatedAt(user.ID)
	if err != nil {
		return nil, false, err
	}

	return user, new, nil
}

// CalibrateUser takes a twitter authenticated user and makes sure it has
// been properly calibrated in the database with all proper keypairs
func CalibrateUser(ta *TruAPI, twitterUser *twitter.User, referrerCode string) (user *db.User, new bool, err error) {
	ctx := context.Background()
	connectedAccount, err := ta.DBClient.ConnectedAccountByTypeAndID("twitter", fmt.Sprintf("%d", twitterUser.ID))
	if err != nil {
		return nil, false, err
	}

	// we'll make a local copy of their avatar photo to remove the dependency on twitter
	avatarURL, err := cacheAvatarLocally(ta.APIContext, twitterUser.ProfileImageURL)
	if err != nil {
		return nil, false, err
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
				AvatarURL: avatarURL,
			},
		}

		user, err := ta.DBClient.AddUserViaConnectedAccount(connectedAccount, referrerCode)
		if err != nil {
			return nil, false, err
		}

		// generating a signing keypair for the user,
		// if they don't have it yet
		if user.Address == "" {
			new = true
			keyPair, err := makeNewKeyPair()
			if err != nil {
				return nil, false, err
			}
			keyPair.UserID = user.ID

			// registering on the chain
			pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
			if err != nil {
				return nil, false, err
			}
			address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
			if err != nil {
				return nil, false, err
			}

			// adding the keypair in the database
			err = ta.DBClient.Add(keyPair)
			if err != nil {
				return nil, false, err
			}
			// adding the address to the user
			err = ta.DBClient.AddAddressToUser(user.ID, address.String())
			if err != nil {
				return nil, false, err
			}
			// follow all communities by default
			communities := ta.communitiesResolver(ctx)
			communityIDs := make([]string, 0)
			for _, community := range communities {
				communityIDs = append(communityIDs, community.ID)
			}
			err = ta.DBClient.FollowCommunities(address.String(), communityIDs)
			if err != nil {
				return nil, false, err
			}
		}

		// if the user has email, we'll subscribe them to the drip emails
		if user.Email != "" {
			// simply logging the error as failure of this API call should not critically fail the registration request
			err = ta.Dripper.ToWorkflow("onboarding").Subscribe(user.Email)
			if err != nil {
				log.Println(err)
			}
		}
	} else {
		user, err := ta.DBClient.UserByID(connectedAccount.UserID)
		if err != nil {
			return nil, false, err
		}
		user.AvatarURL = avatarURL
		err = ta.DBClient.UpdateModel(user)
		if err != nil {
			return nil, false, err
		}
		// this user is already our user, so, we'll just update their meta fields to stay updated
		connectedAccount.Meta = db.ConnectedAccountMeta{
			Email:     twitterUser.Email,
			Bio:       twitterUser.Description,
			Username:  twitterUser.ScreenName,
			FullName:  twitterUser.Name,
			AvatarURL: avatarURL,
		}
		err = ta.DBClient.UpsertConnectedAccount(connectedAccount)
		if err != nil {
			return nil, false, err
		}
	}

	// finally fetching a fresh copy of the user and returning back
	user, err = ta.DBClient.UserByConnectedAccountTypeAndID(connectedAccount.AccountType, connectedAccount.AccountID)
	if err != nil {
		return nil, false, err
	}

	return user, new, nil
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

func cacheAvatarLocally(apiCtx truCtx.TruAPIContext, avatarURL string) (string, error) {
	avatarURL = strings.Replace(avatarURL, "_normal", "_400x400", 1)

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	avatarResponse, err := httpClient.Get(avatarURL)
	if err != nil {
		return "", nil
	}
	defer avatarResponse.Body.Close()

	filename, err := makeFileName(avatarResponse)
	if err != nil {
		return "", nil
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(apiCtx.Config.AWS.S3Region),
		Credentials: credentials.NewStaticCredentials(apiCtx.Config.AWS.AccessKey, apiCtx.Config.AWS.AccessSecret, ""),
	})
	if err != nil {
		return "", err
	}
	contentType := avatarResponse.Header.Get("Content-Type")

	uploader := s3manager.NewUploader(session)
	uploaded, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(apiCtx.Config.AWS.S3Bucket),
		Key:         aws.String(fmt.Sprintf("images/avatar-%s", filename)),
		Body:        avatarResponse.Body,
		ContentType: &contentType,
	})
	if err != nil {
		return "", err
	}

	return uploaded.Location, nil

}

func makeFileName(response *http.Response) (string, error) {
	random := make([]byte, 8)
	_, err := rand.Read(random)
	if err != nil {
		return "", err
	}

	contentType := strings.Split(response.Header.Get("Content-Type"), "/")
	if len(contentType) < 2 {
		return "", nil
	}

	return fmt.Sprintf("%s-%d.%s", hex.EncodeToString(random), time.Now().Unix(), contentType[1]), nil
}
