package truapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/TruStory/octopus/services/truapi/truapi/regex"

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

// RegisterUserRequest represents the schema of the http request to create a new user
type RegisterUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Username  string `json:"username"`
}

// VerifyUserViaTokenRequest updates a user via one-time use token
type VerifyUserViaTokenRequest struct {
	ID    int64  `json:"id"`
	Token string `json:"token"`
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
	case http.MethodPost:
		return ta.createNewUser(r)
	case http.MethodPut:
		return ta.updateUserDetails(r)
	default:
		return ta.getUserDetails(r)
	}
}

func (ta *TruAPI) createNewUser(r *http.Request) chttp.Response {
	var request RegisterUserRequest

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	err = validateRegisterRequest(request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusBadRequest, err)
	}

	user := &db.User{
		FirstName: request.FirstName,
		LastName:  request.LastName,
		Email:     request.Email,
		Password:  request.Password,
		Username:  request.Username,
	}

	err = ta.DBClient.SignupUser(user)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	return chttp.SimpleResponse(http.StatusOK, nil)
}

func (ta *TruAPI) updateUserDetails(r *http.Request) chttp.Response {
	// There are two scenarios when a user can be updated:
	// a.) When users are verifying their email address (via token)
	// b.) When users are updating their profile (bio, photo, etc) from their account settings
	// To update a user, either the user cookie has to be present (for [scenario b]),
	// or a token has to be present (for [scenario a]).

	// attempt to get the cookie
	_, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		// no cookie present; proceed via token
		return ta.verifyUserViaToken(r)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// cookie found, proceed via cookie
	return ta.updateUserDetailsViaCookie(r)
}

func (ta *TruAPI) verifyUserViaToken(r *http.Request) chttp.Response {
	var request VerifyUserViaTokenRequest
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}
	err = json.Unmarshal(reqBody, &request)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	err = ta.DBClient.VerifyUser(request.ID, request.Token)
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
	}

	// MILESTONE -- successfully verified, let's give the user an address (registering user on the chain)
	keyPair, err := makeNewKeyPair()
	if err != nil {
		return chttp.SimpleErrorResponse(http.StatusInternalServerError, err)
	}
	// registering the keypair
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
		if request.Password.New != request.Password.NewConfirmation {
			return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, errors.New("new passwords do not match"))
		}

		err = validatePassword(request.Password.New)
		if err != nil {
			return chttp.SimpleErrorResponse(http.StatusUnprocessableEntity, err)
		}
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

func validateRegisterRequest(request RegisterUserRequest) error {
	request.FirstName = strings.TrimSpace(request.FirstName)
	request.LastName = strings.TrimSpace(request.LastName)
	request.Email = strings.TrimSpace(request.Email)
	request.Username = strings.TrimSpace(request.Username)
	request.Password = strings.TrimSpace(request.Password)

	if request.FirstName == "" {
		return errors.New("first name cannot be empty")
	}

	if request.LastName == "" {
		return errors.New("last name cannot be empty")
	}

	if request.Email == "" {
		return errors.New("email cannot be empty")
	}
	if !regex.IsValidEmail(request.Email) {
		return errors.New("invalid email provided")
	}

	if request.Username == "" {
		return errors.New("username cannot be empty")
	}
	if !regex.IsValidUsername(request.Username) {
		return errors.New("usernames can only contain alphabets, numbers and underscore")
	}
	// https://play.golang.org/p/DxRDseAtacL
	if regex.HasTrustory(request.Username) {
		return errors.New("usernames cannot seem to be related to trustory")
	}

	err := validatePassword(request.Password)
	if err != nil {
		return err
	}

	return nil
}

func validatePassword(password string) error {
	hasMinLength, hasUppercaseLetter, hasLowercaseLetter, hasNumber, hasSpecial := false, false, false, false, false

	for _, char := range password {
		switch {
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsUpper(char):
			hasUppercaseLetter = true
		case unicode.IsLower(char):
			hasLowercaseLetter = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if len(password) >= 8 {
		hasMinLength = true
	}

	if !hasMinLength {
		return errors.New("password must be 8 characters long")
	}

	if !hasNumber {
		return errors.New("password must have a number")
	}

	if !hasUppercaseLetter {
		return errors.New("password must have an uppercase letter")
	}

	if !hasLowercaseLetter {
		return errors.New("password must have a lowercase letter")
	}

	if !hasSpecial {
		return errors.New("password must have a special character")
	}

	return nil
}
