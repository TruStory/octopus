package db

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/TruStory/octopus/services/truapi/truapi/regex"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-pg/pg"
)

// User is the user on the TruStory platform
type User struct {
	Timestamps

	ID         uint64    `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Bio        string    `json:"bio"`
	AvatarURL  string    `json:"avatar_url"`
	Address    string    `json:"address"`
	Password   string    `json:"-"`
	InvitedBy  string    `json:"invited_by"`
	Token      string    `json:"-"`
	ApprovedAt time.Time `json:"approved_at"`
	RejectedAt time.Time `json:"rejected_at"`
	VerifiedAt time.Time `json:"verified_at"`
}

// UserProfile contains the fields that make up the user profile
type UserProfile struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UserPassword contains the fields that allows users to update their passwords
type UserPassword struct {
	Current         string `json:"current"`
	New             string `json:"new"`
	NewConfirmation string `json:"new_confirmation"`
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

// UserByEmail returns the verified user using email
func (c *Client) UserByEmail(email string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("email = ?", email).
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

// UserByUsername returns the verified user using username
func (c *Client) UserByUsername(username string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("username = ?", username).
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

// VerifiedUserByID returns the verified user by ID
func (c *Client) VerifiedUserByID(id uint64) (*User, error) {
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

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("no such user found")
	}

	return user, nil
}

// SignupUser signs up a new user
func (c *Client) SignupUser(user *User) error {
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
	user.Token = base64.StdEncoding.EncodeToString(token)
	user.ApprovedAt = time.Now()

	err = c.AddUser(user)
	if err != nil {
		return err
	}

	return nil
}

// VerifyUser verifies the user via token
func (c *Client) VerifyUser(id uint64, token string) error {
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
func (c *Client) AddAddressToUser(id uint64, address string) error {
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
func (c *Client) ResetPassword(id uint64, password string) error {
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
func (c *Client) UpdatePassword(id uint64, password *UserPassword) error {
	user, err := c.VerifiedUserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}

	if password.New == "" {
		return errors.New("invalid new password")
	}

	if password.New != password.NewConfirmation {
		return errors.New("new passwords do not match")
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
func (c *Client) UpdateProfile(id uint64, profile *UserProfile) error {
	user, err := c.VerifiedUserByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}

	if profile.FirstName == "" {
		return errors.New("first name cannot be left blank")
	}

	if profile.LastName == "" {
		return errors.New("last name cannot be left blank")
	}

	_, err = c.Model(user).
		Where("id = ?", id).
		Where("verified_at IS NOT NULL").
		Where("deleted_at IS NULL").
		Set("first_name = ?", profile.FirstName).
		Set("last_name = ?", profile.LastName).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// ApproveUserByID approves a user to signup (set their password + username)
func (c *Client) ApproveUserByID(id uint64) error {
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
func (c *Client) RejectUserByID(id uint64) error {
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
	_, err := c.Model(user).
		Where("email = ?", user.Email).
		WhereOr("username = ?", user.Username).
		OnConflict("DO NOTHING").
		SelectOrInsert()

	return err
}

// InvitedUsers returns all the users who are invited
func (c *Client) InvitedUsers() ([]User, error) {
	var invitedUsers = make([]User, 0)
	err := c.Model(&invitedUsers).
		Where("invited_by IS NOT NULL").
		Where("deleted_at IS NULL").
		Select()
	if err != nil {
		return invitedUsers, err
	}

	return invitedUsers, nil
}

// InvitedUsersByAddress returns all the users who are invited by a particular address
func (c *Client) InvitedUsersByAddress(address string) ([]User, error) {
	var invitedUsers = make([]User, 0)
	err := c.Model(&invitedUsers).
		Where("invited_by IS NOT NULL").
		Where("deleted_at IS NULL").
		Where("invited_by = ?", address).
		Select()
	if err != nil {
		return invitedUsers, err
	}

	return invitedUsers, nil
}

// AddUserViaConnectedAccount adds a new user using a new connected account
func (c *Client) AddUserViaConnectedAccount(connectedAccount *ConnectedAccount) (*User, error) {
	user := &User{
		FirstName:  connectedAccount.Meta.FullName,
		Username:   connectedAccount.Meta.Username,
		Email:      connectedAccount.Meta.Email,
		Bio:        connectedAccount.Meta.Bio,
		AvatarURL:  connectedAccount.Meta.AvatarURL,
		ApprovedAt: time.Now(),
	}
	err := c.AddUser(user)
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
