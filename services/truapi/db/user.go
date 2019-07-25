package db

// User is the user on the TruStory platform
type User struct {
	Timestamps

	ID           uint64 `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	Password     string `json:"password"`
	InvitedBy    string `json:"invited_by"`
	RequestToken string `json:"request_token"`
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
