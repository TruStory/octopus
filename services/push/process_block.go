package main

import (
	"fmt"
	"strings"

	"github.com/TruStory/octopus/services/truapi/db"
	"github.com/TruStory/truchain/x/account"
	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/types"
)

// Copied from truchain/truapi until truapi is moved into Octopus
func humanReadable(coin sdk.Coin) string {
	// empty struct
	if (sdk.Coin{}) == coin {
		return "0"
	}
	shanevs := sdk.NewDecFromIntWithPrec(coin.Amount, 9).String()
	parts := strings.Split(shanevs, ".")
	number := parts[0]
	decimal := parts[1]
	// If greater than 1.0 => show two decimal digits, truncate trailing zeros
	displayDecimalPlaces := 2
	if number == "0" {
		// If less than 1.0 => show four decimal digits, truncate trailing zeros
		displayDecimalPlaces = 4
	}
	decimal = strings.TrimRight(decimal, "0")
	numberOfDecimalPlaces := len(decimal)
	if numberOfDecimalPlaces > displayDecimalPlaces {
		numberOfDecimalPlaces = displayDecimalPlaces
	}
	decimal = decimal[0:numberOfDecimalPlaces]
	decimal = strings.TrimRight(decimal, "0")
	if decimal == "" {
		return number
	}
	return fmt.Sprintf("%s%s%s", number, ".", decimal)
}

func (s *service) processExpiredStakes(data []byte, notifications chan<- *Notification) {
	expiredStakes := make([]staking.Stake, 0)
	err := staking.ModuleCodec.UnmarshalJSON(data, &expiredStakes)
	if err != nil {
		s.log.WithError(err).Error("error decoding expired stakes")
		return
	}
	for _, expiredStake := range expiredStakes {
		if expiredStake.Result == nil {
			s.log.Errorf("stake result is nil for stake id %d", expiredStake.ID)
			return
		}
		argument, err := s.getClaimArgument(int64(expiredStake.ArgumentID))
		if err != nil {
			s.log.WithError(err).Error("error getting argument ")
			return
		}
		meta := db.NotificationMeta{
			ClaimID:    &argument.ClaimArgument.ClaimID,
			ArgumentID: uint64Ptr(expiredStake.ArgumentID),
		}
		if expiredStake.Result.Type == staking.RewardResultArgumentCreation {
			notifications <- &Notification{
				To: expiredStake.Creator.String(),
				Msg: fmt.Sprintf("You just earned %s %s from your Argument on Claim: %s",
					humanReadable(expiredStake.Result.ArgumentCreatorReward), db.CoinDisplayName,
					argument.ClaimArgument.Claim.Body),
				TypeID: int64(expiredStake.ArgumentID),
				Type:   db.NotificationEarnedStake,
				Meta:   meta,
				Action: fmt.Sprintf("Earned %s", db.CoinDisplayName),
			}
			return
		}
		notifications <- &Notification{
			To: expiredStake.Result.ArgumentCreator.String(),
			Msg: fmt.Sprintf("You just earned %s %s because someone Agreed with you",
				humanReadable(expiredStake.Result.ArgumentCreatorReward), db.CoinDisplayName,
			),
			TypeID: int64(expiredStake.ArgumentID),
			Type:   db.NotificationEarnedStake,
			Meta:   meta,
			Action: fmt.Sprintf("Earned %s", db.CoinDisplayName),
		}
		notifications <- &Notification{
			To: expiredStake.Result.StakeCreator.String(),
			Msg: fmt.Sprintf("You just earned %s %s on an argument you agreed on",
				humanReadable(expiredStake.Result.StakeCreatorReward), db.CoinDisplayName,
			),
			TypeID: int64(expiredStake.ArgumentID),
			Type:   db.NotificationEarnedStake,
			Meta:   meta,
			Action: fmt.Sprintf("Earned %s", db.CoinDisplayName),
		}
	}
}

func (s *service) processUnjailedAccounts(data []byte, notifications chan<- *Notification) {
	unjailed := make([]string, 0)
	err := account.ModuleCodec.UnmarshalJSON(data, &unjailed)
	if err != nil {
		s.log.WithError(err).Error("error decoding unjailed accounts")
		return
	}

	for _, acc := range unjailed {
		notifications <- &Notification{
			To:     acc,
			Msg:    "Hooray you got out of jail!",
			Type:   db.NotificationUnjailed,
			Action: "Unjailed",
		}
	}
}

func (s *service) processBlockEvent(blockEvt types.EventDataNewBlock, notifications chan<- *Notification) {
	for _, event := range blockEvt.ResultEndBlock.Events {
		fmt.Println(event.String())
		switch eventType := event.Type; eventType {
		case account.EventTypeUnjailedAccount:
			for _, attr := range event.GetAttributes() {
				if string(attr.Key) == account.AttributeKeyUser {
					s.processUnjailedAccounts(attr.Value, notifications)
				}
			}
		case staking.EventTypeInterestRewardPaid:
			for _, attr := range event.GetAttributes() {
				if string(attr.Key) == staking.AttributeKeyExpiredStakes {
					s.processExpiredStakes(attr.Value, notifications)
				}
			}
		}
	}
}
