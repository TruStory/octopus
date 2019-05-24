package db

// InitialStakeBalance is the db model to interact with the initial stake balance
type InitialStakeBalance struct {
	Address        string `json:"address"`
	InitialBalance uint64 `json:"initial_balance"  sql:"type:,notnull"`
}

// InitialStakeBalanceByAddress gets the initial stake balance for a user
func (c *Client) InitialStakeBalanceByAddress(address string) (*InitialStakeBalance, error) {
	balance := new(InitialStakeBalance)
	err := c.Model(&balance).Where("address = ?", address).Select()

	if err != nil {
		return nil, err
	}

	return balance, nil
}

// UpsertInitialStakeBalance inserts or updates the initial stake balance
func (c *Client) UpsertInitialStakeBalance(balance InitialStakeBalance) error {
	_, err := c.Model(&balance).
		OnConflict("(address) DO UPDATE").
		Set("address = EXCLUDED.address, initial_balance = EXCLUDED.initial_balance").
		Insert()

	return err
}
