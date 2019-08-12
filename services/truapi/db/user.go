package db

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/regex"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-pg/pg"
)

// InvitedUserDefaultName is the default name given to the invited user
const InvitedUserDefaultName = "<invited user>"

// User is the user on the TruStory platform
type User struct {
	Timestamps

	ID                  int64      `json:"id"`
	FullName            string     `json:"full_name"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	Bio                 string     `json:"bio"`
	AvatarURL           string     `json:"avatar_url"`
	Address             string     `json:"address"`
	Password            string     `json:"-" graphql:"-"`
	ReferredBy          int64      `json:"referred_by"`
	Token               string     `json:"-" graphql:"-"`
	ApprovedAt          time.Time  `json:"approved_at" graphql:"-"`
	RejectedAt          time.Time  `json:"rejected_at" graphql:"-"`
	VerifiedAt          time.Time  `json:"verified_at" graphql:"-"`
	BlacklistedAt       time.Time  `json:"blacklisted_at" graphql:"-"`
	LastAuthenticatedAt *time.Time `json:"last_authenticated_at" graphql:"-"`
}

// UserProfile contains the fields that make up the user profile
type UserProfile struct {
	FullName  string `json:"full_name"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
}

// UserPassword contains the fields that allows users to update their passwords
type UserPassword struct {
	Current         string `json:"current"`
	New             string `json:"new"`
	NewConfirmation string `json:"new_confirmation"`
}

// UserCredentials contains the fields that allows users to log into their accounts
type UserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserByID selects a user by ID
func (c *Client) UserByID(ID int64) (*User, error) {
	user := new(User)
	err := c.Model(user).Where("id = ?", ID).First()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UserByEmailOrUsername selects a user either by email or username
func (c *Client) UserByEmailOrUsername(identifier string) (*User, error) {
	if regex.IsValidEmail(identifier) {
		return c.UserByEmail(identifier)
	}

	if regex.IsValidUsername(identifier) {
		return c.UserByUsername(identifier)
	}

	return nil, errors.New("no such user")
}

// UserByEmail returns the user using email
func (c *Client) UserByEmail(email string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("LOWER(email) = ?", strings.ToLower(email)).
		Where("deleted_at IS NULL").
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UserByUsername returns the user using username
func (c *Client) UserByUsername(username string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("LOWER(username) = ?", strings.ToLower(username)).
		Where("deleted_at IS NULL").
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UserByAddress returns the verified user using address
func (c *Client) UserByAddress(address string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("address = ?", address).
		Where("deleted_at IS NULL").
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// VerifiedUserByID returns the verified user by ID
func (c *Client) VerifiedUserByID(id int64) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("id = ?", id).
		Where("verified_at IS NOT NULL").
		Where("deleted_at IS NULL").
		First()

	if err == pg.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetAuthenticatedUser authenticates the user and returns the authenticated user
func (c *Client) GetAuthenticatedUser(identifier, password string) (*User, error) {
	user, err := c.UserByEmailOrUsername(identifier)

	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("no such user found")
	}

	if !user.BlacklistedAt.IsZero() {
		return nil, errors.New("the user is blacklisted and cannot be authenticated")
	}

	if user.VerifiedAt.IsZero() {
		return nil, errors.New("the user has not verified their email address yet")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("no such user found")
	}

	err = c.TouchLastAuthenticatedAt(user.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// TouchLastAuthenticatedAt updates the last_authenticated_at column with the current timestamp
func (c *Client) TouchLastAuthenticatedAt(id int64) error {
	var user User
	_, err := c.Model(&user).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("last_authenticated_at = ?", time.Now()).
		Update()
	if err != nil {
		return err
	}
	return nil
}

// SignupUser signs up a new user
func (c *Client) SignupUser(user *User, referrerCode string) error {
	salt, err := generateCryptoSafeRandomBytes(16)
	if err != nil {
		return err
	}
	token, err := generateCryptoSafeRandomBytes(32)
	if err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(salt, []byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.Token = hex.EncodeToString(token)
	user.ApprovedAt = time.Now()

	referrer, err := c.UserByAddress(referrerCode)
	if err != nil {
		return err
	}
	if referrer != nil {
		user.ReferredBy = referrer.ID
	}

	err = c.AddUser(user)
	if err != nil {
		return err
	}

	return nil
}

// VerifyUser verifies the user via token
func (c *Client) VerifyUser(id int64, token string) error {
	var user User
	result, err := c.Model(&user).
		Where("id = ?", id).
		Where("token = ?", token).
		Where("verified_at IS NULL").
		Where("deleted_at IS NULL").
		Set("verified_at = ?", time.Now()).
		Update()
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("invalid token")
	}

	return nil
}

// AddAddressToUser adds a cosmos address to the user
func (c *Client) AddAddressToUser(id int64, address string) error {
	var user User
	_, err := c.Model(&user).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("address = ?", address).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// ResetPassword resets the user's password to a new one
func (c *Client) ResetPassword(id int64, password string) error {
	user, err := c.VerifiedUserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}

	salt := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(salt, []byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = c.Model(user).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Set("password = ?", string(hashedPassword)).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// UpdatePassword changes a password for a user
func (c *Client) UpdatePassword(id int64, password *UserPassword) error {
	user, err := c.VerifiedUserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password.Current))
	if err != nil {
		return errors.New("incorrect current password")
	}

	salt := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(salt, []byte(password.New), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NOT NULL").
		Where("deleted_at IS NULL").
		Set("password = ?", string(hashedPassword)).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// UpdateProfile changes a profile fields for a user
func (c *Client) UpdateProfile(id int64, profile *UserProfile) error {
	user, err := c.VerifiedUserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}

	if profile.FullName == "" {
		return errors.New("name cannot be left blank")
	}

	_, err = c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NOT NULL").
		Where("deleted_at IS NULL").
		Set("full_name = ?", profile.FullName).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// SetUserCredentials adds an email + password combo to an existing user, who was previously authorized via some connected account
func (c *Client) SetUserCredentials(id int64, credentials *UserCredentials) error {
	user, err := c.UserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}
	if !user.VerifiedAt.IsZero() {
		return errors.New("this user already has credentials set and verified")
	}

	hashedPassword, err := getHashedPassword(credentials.Password)
	if err != nil {
		return nil
	}

	_, err = c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NULL").
		Where("deleted_at IS NULL").
		Set("email = ?", credentials.Email).
		Set("password = ?", hashedPassword).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// ApproveUserByID approves a user to signup (set their password + username)
func (c *Client) ApproveUserByID(id int64) error {
	user := new(User)
	_, err := c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NULL"). // the flag can be updated only until the user hasn't signed up
		Set("approved_at = NOW()").
		Set("rejected_at = NULL").
		Update()

	if err != nil {
		return err
	}

	return nil
}

// RejectUserByID rejects a user from signing up (set their password + username)
func (c *Client) RejectUserByID(id int64) error {
	user := new(User)
	_, err := c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NULL"). // the flag can be updated only until the user hasn't signed up
		Set("rejected_at = ?", time.Now()).
		Set("approved_at = NULL").
		Update()

	if err != nil {
		return err
	}

	return nil
}

// AddUser upserts the user into the database
func (c *Client) AddUser(user *User) error {
	user.Email = strings.ToLower(user.Email)
	inserted, err := c.Model(user).
		Where("LOWER(email) = ?", user.Email).
		WhereOr("LOWER(username) = ?", strings.ToLower(user.Username)).
		OnConflict("DO NOTHING").
		SelectOrInsert()

	if !inserted {
		return errors.New("a user already exists with same email/username")
	}

	return err
}

// BlacklistUser blacklists a user and prevents them from logging in
func (c *Client) BlacklistUser(id int64) error {
	var user User
	result, err := c.Model(&user).
		Where("id = ?", id).
		Set("blacklisted_at = ?", time.Now()).
		Update()
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("invalid user")
	}

	return nil
}

// UnblacklistUser unblacklists a user and allows them from logging in again
func (c *Client) UnblacklistUser(id int64) error {
	var user User
	result, err := c.Model(&user).
		Where("id = ?", id).
		Set("blacklisted_at = NULL").
		Update()
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("invalid user")
	}

	return nil
}

// InvitedUsers returns all the users who are invited
func (c *Client) InvitedUsers() ([]User, error) {
	var invitedUsers = make([]User, 0)
	err := c.Model(&invitedUsers).
		Where("referred_by IS NOT NULL").
		Where("deleted_at IS NULL").
		Select()
	if err != nil {
		return invitedUsers, err
	}

	return invitedUsers, nil
}

// InvitedUsersByID returns all the users who are invited by a particular address
func (c *Client) InvitedUsersByID(referrerID int64) ([]User, error) {
	var invitedUsers = make([]User, 0)
	err := c.Model(&invitedUsers).
		Where("deleted_at IS NULL").
		Where("referred_by = ?", referrerID).
		Select()
	if err != nil {
		return invitedUsers, err
	}

	return invitedUsers, nil
}

// AddUserViaConnectedAccount adds a new user using a new connected account
func (c *Client) AddUserViaConnectedAccount(connectedAccount *ConnectedAccount) (*User, error) {
	// a.) check if their email address is associated with an existing account.
	// if yes, merge them with that account
	if connectedAccount.Meta.Email != "" {
		user, err := c.UserByEmail(connectedAccount.Meta.Email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			connectedAccount.UserID = user.ID
			err = c.UpsertConnectedAccount(connectedAccount)
			if err != nil {
				return nil, err
			}

			return user, nil
		}
	}

	// b.) if no existing account found, continue creating a new account
	// (if the their connected account's username is not available on the platform,
	// we'll create a random one for them that they can edit later.)
	username, err := getUniqueUsername(c, connectedAccount.Meta.Username, "")
	if err != nil {
		return nil, err
	}
	user := &User{
		FullName:   connectedAccount.Meta.FullName,
		Username:   username,
		Email:      connectedAccount.Meta.Email,
		Bio:        connectedAccount.Meta.Bio,
		AvatarURL:  connectedAccount.Meta.AvatarURL,
		ApprovedAt: time.Now(),
	}
	err = c.AddUser(user)
	if err != nil {
		return nil, err
	}

	connectedAccount.UserID = user.ID
	err = c.UpsertConnectedAccount(connectedAccount)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UserByConnectedAccountTypeAndID returns the user that has a given connected account
func (c *Client) UserByConnectedAccountTypeAndID(accountType, accountID string) (*User, error) {
	connectedAccount, err := c.ConnectedAccountByTypeAndID(accountType, accountID)
	if err != nil {
		return nil, err
	}

	user := &User{ID: connectedAccount.UserID}
	err = c.Find(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// IsTwitterUser returns a twitter user that has a given connected account
func (c *Client) IsTwitterUser(userID int64) bool {
	connectedAccount, err := c.ConnectedAccountByTypeAndUserID("twitter", userID)
	if err != nil {
		return false
	}
	return connectedAccount != nil
}

func getUniqueUsername(c *Client, username string, suffix string) (string, error) {
	candidate := username + suffix
	user, err := c.UserByUsername(username + suffix)
	if err != nil {
		return "", err
	}
	if user != nil {
		intSuffix := 0
		if suffix != "" {
			intSuffix, err = strconv.Atoi(suffix)
			if err != nil {
				return "", err
			}
		}
		return getUniqueUsername(c, username, strconv.Itoa(intSuffix+1))
	}

	return candidate, nil
}

// UsernamesByPrefix returns the first five usernames for the provided prefix string
func (c *Client) UsernamesByPrefix(prefix string) (usernames []string, err error) {
	var users []User
	sqlFragment := fmt.Sprintf("username ILIKE '%s", prefix)
	err = c.Model(&users).Where(sqlFragment + "%'").Limit(5).Select()
	if err == pg.ErrNoRows {
		return usernames, nil
	}
	if err != nil {
		return usernames, err
	}
	for _, user := range users {
		usernames = append(usernames, user.Username)
	}

	return usernames, nil
}

// UserProfileByAddress fetches user profile details by address
func (c *Client) UserProfileByAddress(addr string) (*UserProfile, error) {
	userProfile := new(UserProfile)
	user := new(User)
	err := c.Model(user).Where("address = ?", addr).Select()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return userProfile, err
	}

	userProfile = &UserProfile{
		FullName:  user.FullName,
		Bio:       user.Bio,
		AvatarURL: user.AvatarURL,
		Username:  user.Username,
	}

	return userProfile, nil
}

// UserProfileByUsername fetches user profile by username
func (c *Client) UserProfileByUsername(username string) (*UserProfile, error) {
	userProfile := new(UserProfile)
	user := new(User)
	err := c.Model(user).Where("username = ?", username).First()
	if err == pg.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return userProfile, err
	}

	userProfile = &UserProfile{
		FullName:  user.FullName,
		Bio:       user.Bio,
		AvatarURL: user.AvatarURL,
		Username:  user.Username,
	}

	return userProfile, nil
}

func getHashedPassword(password string) (string, error) {
	salt, err := generateCryptoSafeRandomBytes(16)
	if err != nil {
		return "", err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(salt, []byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}
