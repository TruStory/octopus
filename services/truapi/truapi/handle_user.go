package truapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/octopus/services/truapi/postman/messages"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/octopus/services/truapi/truapi/regex"
	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// UserResponse is a JSON response body representing the result of User
type UserResponse struct {
	UserID        int64                `json:"userId"`
	Address       string               `json:"address"`
	Bio           string               `json:"bio"`
	InvitesLeft   int64                `json:"invitesLeft"`
	UserMeta      db.UserMeta          `json:"userMeta"`
	UserProfile   *UserProfileResponse `json:"userProfile"`
	Group         string               `json:"group"`
	SignedUp      bool                 `json:"signedUp"`
	AccountNumber uint64               `json:"accountNumber"`
	Sequence      uint64               `json:"sequence"`
}

// VerificationRespose represents the response sent when a users verifies the email
type VerificationRespose struct {
	User     UserResponse `json:"user"`
	Verified bool         `json:"verified"`
	Created  bool         `json:"created"`
}

// UserProfileResponse is a JSON response body representing the UserProfile
type UserProfileResponse struct {
	Username  string `json:"username"`
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarURL"`
	Bio       string `json:"bio"`
}

// RegisterUserRequest represents the schema of the http request to create a new user
type RegisterUserRequest struct {
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Username   string `json:"username"`
	ReferredBy string `json:"referred_by"`
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

	// Credentials fields
	Credentials *db.UserCredentials `json:"credentials,omitempty"`
}

// TruErrors for handle user
var (
	ErrExistingAccountWithEmail = render.TruError{Code: 100, Message: "There's already an account with this email address."}
	ErrExistingTwitterAccount   = render.TruError{Code: 101, Message: "This email is associated with a Twitter account. Please log in with Twitter."}
	ErrEmailNotVerified         = render.TruError{Code: 102, Message: "The account associated with this email is not verified yet."}
	ErrUsernameTaken            = render.TruError{Code: 103, Message: "This username is already taken."}
	ErrCannotSendEmail          = render.TruError{Code: 104, Message: "Error sending email."}
	ErrInvalidPasswords         = render.TruError{Code: 105, Message: "Passwords don't match."}
	ErrInvalidPassword          = render.TruError{Code: 106, Message: "Invalid password."}
	ErrUserNotFound             = render.TruError{Code: 107, Message: "User not found."}
	ErrRegistration             = render.TruError{Code: 108, Message: "Registration error."}
	ErrInvalidEmail             = render.TruError{Code: 109, Message: "Invalid email."}
)

// HandleUserDetails takes a `UserRequest` and returns a `UserResponse`
func (ta *TruAPI) HandleUserDetails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		ta.createNewUser(w, r)
	case http.MethodPut:
		ta.updateUserDetailsViaCookie(w, r)
	default:
		ta.getUserDetails(w, r)
	}
}

func getEmailDomain(email string) string {
	e := strings.TrimSpace(email)
	if e == "" {
		return ""
	}
	parts := strings.Split(e, "@")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

func (ta *TruAPI) createNewUser(w http.ResponseWriter, r *http.Request) {
	var request RegisterUserRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// ensure email is lowercase
	request.Email = strings.ToLower(request.Email)

	err = validateRegisterRequest(request)
	if err != nil {
		render.LoginError(
			w, r,
			render.TruError{Code: ErrRegistration.Code, Message: err.Error()},
			http.StatusBadRequest)

		return
	}
	// check domain is in whitelist
	whitelisted, err := ta.DBClient.IsDomainWhitelisted(getEmailDomain(request.Email))
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	if !whitelisted {
		log.Println("Registration failed because domain is not in the whitelist ", request.Email)
		render.LoginError(
			w, r,
			render.TruError{Code: ErrInvalidEmail.Code, Message: ErrInvalidEmail.Error()},
			http.StatusBadRequest)

		return
	}

	user, err := ta.DBClient.UserByEmail(request.Email)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	if user != nil {
		// if a valid and verified account is already existing for this email, we'll send back an error
		if !user.VerifiedAt.IsZero() {
			render.LoginError(w, r, ErrExistingAccountWithEmail, http.StatusBadRequest)
			return
		}

		// if the account with this email address is usually logged in via Twitter, we'll let them know
		connectedAccounts, err := ta.DBClient.ConnectedAccountsByUserID(user.ID)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(connectedAccounts) > 0 {
			render.LoginError(w, r, ErrExistingTwitterAccount, http.StatusBadRequest)
			return
		}

		// if the account with this email address is simply pending verification, we'll let them know
		render.LoginError(w, r, ErrEmailNotVerified, http.StatusBadRequest)
		return
	}

	user, err = ta.DBClient.UserByUsername(request.Username)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	if user != nil {
		render.LoginError(w, r, ErrUsernameTaken, http.StatusBadRequest)
		return
	}

	user = &db.User{
		FullName: request.FullName,
		Email:    request.Email,
		Password: request.Password,
		Username: request.Username,
	}

	err = ta.DBClient.RegisterUser(user, request.ReferredBy, ta.APIContext.Config.Defaults.AvatarURL)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err = sendVerificationEmail(ta, *user)
	if err != nil {
		render.LoginError(w, r, ErrCannotSendEmail, http.StatusInternalServerError)
		return
	}

	render.Response(w, r, user, http.StatusOK)
}

func (ta *TruAPI) verifyUserViaToken(w http.ResponseWriter, r *http.Request) {
	ctx := ta.createContext(r.Context())
	var request VerifyUserViaTokenRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	cookie := cookies.GetLogoutCookie(ta.APIContext)
	http.SetCookie(w, cookie)
	err = ta.DBClient.VerifyUser(request.ID, request.Token)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := ta.DBClient.VerifiedUserByID(request.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// user is already registered on the chain and has an address
	if user.Address != "" {
		userResponse := ta.createUserResponse(r.Context(), user, false)
		resp := VerificationRespose{
			User:     userResponse,
			Verified: true,
			Created:  false,
		}
		render.Response(w, r, resp, http.StatusOK)
		return
	}

	// successfully verified; if user doesn't have adderss, let's give the user an address (registering user on the chain)
	keyPair, err := makeNewKeyPair()
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	keyPair.UserID = user.ID

	// registering the keypair
	pubKeyBytes, err := hex.DecodeString(keyPair.PublicKey)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	address, err := ta.RegisterKey(pubKeyBytes, "secp256k1")
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// adding the keypair in the database
	err = ta.DBClient.Add(keyPair)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ta.DBClient.AddAddressToUser(request.ID, address.String())
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	// follow all communities by default
	communities := ta.communitiesResolver(ctx)
	communityIDs := make([]string, 0)
	for _, community := range communities {
		communityIDs = append(communityIDs, community.ID)
	}
	err = ta.DBClient.FollowCommunities(address.String(), communityIDs)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// simply logging the error as failure of this API call should not critically fail the registration request
	err = ta.Dripper.ToWorkflow("onboarding").Subscribe(user.Email)
	if err != nil {
		log.Println(err)
	}
	user.Address = address.String()
	userResponse := ta.createUserResponse(r.Context(), user, false)
	resp := VerificationRespose{
		User:     userResponse,
		Verified: true,
		Created:  true,
	}
	render.Response(w, r, resp, http.StatusOK)
}

func (ta *TruAPI) updateUserDetailsViaCookie(w http.ResponseWriter, r *http.Request) {
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}

	var request UpdateUserViaCookieRequest
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	// if user wants to change their password
	if request.Password != nil {
		if request.Password.New != request.Password.NewConfirmation {
			render.LoginError(w, r, ErrInvalidPasswords, http.StatusBadRequest)
			return
		}

		err = validatePassword(request.Password.New)
		if err != nil {
			render.LoginError(w, r, render.TruError{Code: ErrInvalidPassword.Code, Message: err.Error()}, http.StatusOK)
			return
		}
		err = ta.DBClient.UpdatePassword(user.ID, request.Password)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		render.Response(w, r, true, http.StatusOK)
		return
	}

	// if user wants to change their profile
	if request.Profile != nil {
		err = ta.DBClient.UpdateProfile(user.ID, request.Profile)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		render.Response(w, r, true, http.StatusOK)
		return
	}

	// if user (who was previously authorized via connected account) wants to add a password to their accounts
	if request.Credentials != nil {
		err = ta.DBClient.SetUserCredentials(user.ID, request.Credentials)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		userToBeVerified, err := ta.DBClient.UserByID(user.ID)
		if err != nil {
			render.Error(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if userToBeVerified == nil {
			//  this is a redundant check. it will never be trigger as invalid ID will be handled by SetUserCredentials method already.
			// it is here in the rare case of something changing during refactoring sometime in the future.
			render.Error(w, r, "the user cannot be verified right now", http.StatusInternalServerError)
			return
		}
		err = sendVerificationEmail(ta, *userToBeVerified)
		if err != nil {
			render.Error(w, r, "cannot send email confirmation right now", http.StatusInternalServerError)
			return
		}

		render.Response(w, r, true, http.StatusOK)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}

func (ta *TruAPI) getUserDetails(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}
	user, err := ta.DBClient.UserByID(authenticatedUser.ID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusUnauthorized)
		return
	}
	if user == nil {
		render.LoginError(w, r, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	response := ta.createUserResponse(r.Context(), user, false)
	render.Response(w, r, response, http.StatusOK)
}

func (ta *TruAPI) createUserResponse(ctx context.Context, user *db.User, singedUp bool) UserResponse {
	largeURI := strings.Replace(user.AvatarURL, "_bigger", "_200x200", 1)
	largeURI = strings.Replace(largeURI, "http://", "https://", 1)

	aa := ta.appAccountResolver(ctx, queryByAddress{ID: user.Address})
	var accountNumber, sequence uint64
	if aa != nil {
		accountNumber = aa.AccountNumber
		sequence = aa.Sequence
	}
	return UserResponse{
		UserID:      user.ID,
		Address:     user.Address,
		Bio:         user.Bio,
		InvitesLeft: user.InvitesLeft,
		UserMeta:    user.Meta,
		Group:       user.UserGroup.String(),
		UserProfile: &UserProfileResponse{
			Bio:       user.Bio,
			AvatarURL: largeURI,
			FullName:  user.FullName,
			Username:  user.Username,
		},
		SignedUp:      singedUp,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}
}

func validateRegisterRequest(request RegisterUserRequest) error {
	request.FullName = strings.TrimSpace(request.FullName)
	request.Email = strings.TrimSpace(request.Email)
	request.Username = strings.TrimSpace(request.Username)
	request.Password = strings.TrimSpace(request.Password)

	if request.FullName == "" {
		return errors.New("first name cannot be empty")
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

func sendVerificationEmail(ta *TruAPI, user db.User) error {
	message, err := messages.MakeEmailConfirmationMessage(ta.Postman, ta.APIContext.Config, user)
	if err != nil {
		return err
	}

	err = ta.Postman.Deliver(*message)
	if err != nil {
		return err
	}

	return nil
}
