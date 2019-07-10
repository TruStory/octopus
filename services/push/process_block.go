package main

import (
	"fmt"
	"strings"

	"github.com/TruStory/truchain/x/staking"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/types"

	"github.com/TruStory/octopus/services/truapi/db"
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

func (s *service) processBlockEvent(blockEvt types.EventDataNewBlock, notifications chan<- *Notification) {
	for _, tag := range blockEvt.ResultEndBlock.Tags {

		if string(tag.Key) == "expired-stakes" {
			expiredStakes := make([]staking.Stake, 0)
			err := staking.ModuleCodec.UnmarshalJSON(tag.Value, &expiredStakes)
			if err != nil {
				s.log.WithError(err).Error("error decoding expired stakes")
				continue
			}
			for _, expiredStake := range expiredStakes {
				if expiredStake.Result == nil {
					s.log.Errorf("stake result is nil for stake id %d", expiredStake.ID)
					continue
				}
				argument, err := s.getClaimArgument(int64(expiredStake.ArgumentID))
				if err != nil {
					s.log.WithError(err).Error("error getting argument ")
					continue
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
						Action: "Earned TruStake",
					}
					continue
				}
				notifications <- &Notification{
					To: expiredStake.Result.ArgumentCreator.String(),
					Msg: fmt.Sprintf("You just earned %s %s on an argument someone agreed on",
						humanReadable(expiredStake.Result.ArgumentCreatorReward), db.CoinDisplayName,
					),
					TypeID: int64(expiredStake.ArgumentID),
					Type:   db.NotificationEarnedStake,
					Meta:   meta,
					Action: "Earned TruStake",
				}
				notifications <- &Notification{
					To: expiredStake.Result.StakeCreator.String(),
					Msg: fmt.Sprintf("You just earned %s %s on an argument you agreed on",
						humanReadable(expiredStake.Result.StakeCreatorReward), db.CoinDisplayName,
					),
					TypeID: int64(expiredStake.ArgumentID),
					Type:   db.NotificationEarnedStake,
					Meta:   meta,
					Action: "Earned TruStake",
				}
			}
		}
	}
}
