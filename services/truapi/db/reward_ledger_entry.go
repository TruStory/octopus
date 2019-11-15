package db

// RewardLedgerEntry represents an entry into the reward ledger
type RewardLedgerEntry struct {
	Timestamps

	ID        int64                      `json:"id"`
	UserID    int64                      `json:"user_id"`
	Direction RewardLedgerEntryDirection `json:"direction"`
	Amount    int64                      `json:"amount"`
	Currency  RewardLedgerEntryCurrency  `json:"currency"`
}

// RewardLedgerEntryDirection represents the direction for an entry
type RewardLedgerEntryDirection string

const (
	RewardLedgerEntryDirectionCredit RewardLedgerEntryDirection = "credit"
	RewardLedgerEntryDirectionDebit  RewardLedgerEntryDirection = "debit"
)

// RewardLedgerEntryCurrency represents the currency for an entry
type RewardLedgerEntryCurrency string

const (
	RewardLedgerEntryCurrencyInvite RewardLedgerEntryCurrency = "invite"
	RewardLedgerEntryCurrencyTru    RewardLedgerEntryCurrency = "utru"
)

// RecordRewardLedgerEntry makes and record an entry in the reward ledger
func (c *Client) RecordRewardLedgerEntry(
	userID int64,
	direction RewardLedgerEntryDirection,
	amount int64,
	currency RewardLedgerEntryCurrency,
) (*RewardLedgerEntry, error) {

	entry := &RewardLedgerEntry{
		UserID:    userID,
		Direction: direction,
		Amount:    amount,
		Currency:  currency,
	}

	err := c.Add(entry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
