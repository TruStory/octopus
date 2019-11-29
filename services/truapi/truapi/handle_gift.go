package truapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/db"
	app "github.com/TruStory/truchain/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/TruStory/octopus/services/truapi/truapi/render"
)

// GiftRequest represents the request to gift TRU to users
type GiftRequest struct {
	UserID int64  `json:"user_id"`
	Amount string `json:"amount"`
	Memo   string `json:"memo"`
}

// HandleGift gifts TRU to the user
func (ta *TruAPI) HandleGift(w http.ResponseWriter, r *http.Request) {
	// only supports GET requests
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request GiftRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := ta.DBClient.UserByID(request.UserID)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	amount, err := sdk.ParseCoin(request.Amount)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if amount.Denom != app.StakeDenom {
		err = fmt.Errorf("invalid denomination coin got %s wanted %s", amount.Denom, app.StakeDenom)
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	broker, err := ta.accountQuery(r.Context(), ta.APIContext.Config.RewardBroker.Addr)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
	err = ta.SendGiftToAddress(user.Address, amount, broker.GetAccountNumber(), broker.GetSequence(), request.Memo)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = ta.DBClient.RecordRewardLedgerEntry(user.ID, db.RewardLedgerEntryDirectionCredit, amount.Amount.Int64(), db.RewardLedgerEntryCurrencyTru)
	if err != nil {
		render.Error(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	render.Response(w, r, true, http.StatusOK)
}
