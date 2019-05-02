package main

// User defines the schema that represents a user
type User struct {
	UserID               string `json:"userId"`
	Username             string `json:"username"`
	FullName             string `json:"fullname"`
	Address              string `json:"address"`
	AuthenticationCookie string `json:"authenticationCookie"`
}

// MockedRegisterResponse represents a newly registeres mock user
type MockedRegisterResponse struct {
	Data User `json:"data"`
}

// Amount defines the schema of representing an amount
type Amount struct {
	Denomination string `json:"denom"`
	Amount       string `json:"amount"`
}

// Fee defines the schema of the fee to be paid
type Fee struct {
	Amount []Amount `json:"amount"`
	Gas    int64    `json:"gas"`
}

// SubmitStoryMsg defines the schema of the msg type SubmitStoryMsg
type SubmitStoryMsg struct {
	Creator    string `json:"creator"`
	Body       string `json:"body"`
	CategoryID int64  `json:"category_id"`
	StoryType  int64  `json:"story_type"`
}

// AddStoryRequest defines the schema of the Tx
type AddStoryRequest struct {
	AccountNumber int64            `json:"account_number"`
	ChainID       string           `json:"chain_id"`
	Fee           Fee              `json:"fee"`
	Memo          string           `json:"memo"`
	Msgs          []SubmitStoryMsg `json:"msgs"`
	Sequence      int64            `json:"sequence"`
}

// UnsignedRequest defines the schema of the unsigned tx
type UnsignedRequest struct {
	MsgTypes []string `json:"msg_types"`
	Tx       string   `json:"tx"`
	TxRaw    string   `json:"tx_raw"`
}
