package db

import (
	"crypto/rand"
	"errors"
	"io"
	"time"

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
	Address    string    `json:"address"`
	Password   string    `json:"password"`
	InvitedBy  string    `json:"invited_by"`
	Token      string    `json:"token"`
	ApprovedAt time.Time `json:"approved_at"`
	RejectedAt time.Time `json:"rejected_at"`
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

// UserByEmail returns the signedup user using email
func (c *Client) UserByEmail(email string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("email = ?", email).
		Where("signedup_at IS NOT NULL").
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

// UserByUsername returns the signedup user using username
func (c *Client) UserByUsername(username string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("username = ?", username).
		Where("signedup_at IS NOT NULL").
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

// SignedupUserByID returns the signedup user by ID
func (c *Client) SignedupUserByID(id uint64) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("id = ?", id).
		Where("signedup_at IS NOT NULL").
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

// UnsignedupUserByIDAndToken returns the unsignedup user using the combination of id and request_token
func (c *Client) UnsignedupUserByIDAndToken(id uint64, token string) (*User, error) {
	var user User
	err := c.Model(&user).
		Where("id = ?", id).
		Where("token = ?", token).
		Where("signedup_at IS NULL").
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
func (c *Client) GetAuthenticatedUser(email, username, password string) (*User, error) {
	var user *User
	var err error
	if email != "" {
		// if email is present, we'll first attempt with email
		user, err = c.UserByEmail(email)
	} else if username != "" {
		// then, we'll attempt with username
		user, err = c.UserByUsername(username)
	}
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

// SignupUser signs up a user by setting the username and a password
func (c *Client) SignupUser(id uint64, token string, username string, password string) error {
	user, err := c.UnsignedupUserByIDAndToken(id, token)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("no such user found")
	}
	if user.ApprovedAt.IsZero() {
		return errors.New("user is not approved for signing up")
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
		Where("token = ?", token).
		Where("signedup_at IS NULL").
		Where("deleted_at IS NULL").
		Set("username = ?", username).
		Set("password = ?", string(hashedPassword)).
		Set("signedup_at = ?", time.Now()).
		Update()

	if err != nil {
		return err
	}

	return nil
}

// UpdatePassword changes a password for a user
func (c *Client) UpdatePassword(id uint64, password *UserPassword) error {
	user, err := c.SignedupUserByID(id)
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
		Where("signedup_at IS NOT NULL").
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
	user, err := c.SignedupUserByID(id)
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
		Where("signedup_at IS NOT NULL").
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
		Where("signedup_at IS NULL"). // the flag can be updated only until the user hasn't signed up
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
		Where("signedup_at IS NULL"). // the flag can be updated only until the user hasn't signed up
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
		Where("username = ?", user.Username).
		OnConflict("DO NOTHING").
		SelectOrInsert()

	return err
}
