package main

import (
	"encoding/json"

	"fmt"

	truchain "github.com/TruStory/truchain/types"
	db "github.com/TruStory/truchain/x/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/types"
)

func getTrustakeValue(c sdk.Coin) sdk.Int {
	return sdk.NewDecFromInt(c.Amount).QuoInt(sdk.NewInt(truchain.Shanev)).RoundInt()
}
func getWinnerMsg(stake, reward, interest sdk.Coin) string {
	s, r, i := getTrustakeValue(stake), getTrustakeValue(reward), getTrustakeValue(interest)
	// case when you were the only staker
	if reward.IsZero() {
		msg := fmt.Sprintf("You were refunded %s TruStake", s)
		if i.IsZero() {
			return msg
		}
		return fmt.Sprintf("%s and earned %s interest", msg, i)
	}
	msg := fmt.Sprintf("You won %s TrusTake", r)
	if i.IsZero() {
		return msg
	}
	return fmt.Sprintf("%s but earned %s TruStake in interest", msg, i)
}
func getLoserMsg(stake, interest sdk.Coin) string {
	s, i := getTrustakeValue(stake), getTrustakeValue(interest)
	msg := fmt.Sprintf("You lost %s TrusTake", s)
	if i.IsZero() {
		return msg
	}
	return fmt.Sprintf("%s but earned %s TruStake in interest", msg, i)
}

func getResultMessage(t truchain.StakeDistributionResultsType, isBacker bool, staker truchain.Staker, earns UserEarns) string {
	switch t {
	case truchain.DistributionMajorityNotReached:
		i := getTrustakeValue(earns.Interest)
		if i.IsZero() {
			return "It's a tie"
		}
		return fmt.Sprintf("It's a tie but you earned %s interest", getTrustakeValue(earns.Interest))
	case truchain.DistributionBackersWin:
		if isBacker {
			return getWinnerMsg(staker.Amount, earns.Reward, earns.Interest)
		}
		return getLoserMsg(staker.Amount, earns.Interest)
	case truchain.DistributionChallengersWin:
		if isBacker {
			return getLoserMsg(staker.Amount, earns.Interest)
		}
		return getWinnerMsg(staker.Amount, earns.Reward, earns.Interest)

	}
	return ""
}

func (s *service) processNewBlockEvent(newBlockEvent types.EventDataNewBlock, notifications chan<- *Notification) {
	for _, tag := range newBlockEvent.ResultEndBlock.Tags {
		if string(tag.Key) == "tru.event.completedStories" {
			completed := &truchain.CompletedStoriesNotificationResult{}
			err := json.Unmarshal(tag.Value, completed)
			if err != nil {
				s.log.WithError(err).Error("error decoding completed stories")
				continue
			}
			for _, story := range completed.Stories {
				var creatorStaked bool

				usersEarns := processStorySummary(story)
				meta := db.NotificationMeta{
					StoryID: &story.ID,
				}
				for _, backer := range story.Backers {
					if story.Creator.String() == backer.Address.String() {
						creatorStaked = true
					}
					earns, ok := usersEarns[backer.Address.String()]

					if !ok {
						s.log.WithField("address", backer.Address.String()).Info("earns not found")
					}

					msg := getResultMessage(story.StakeDistributionResults.Type, true, backer, earns)
					notifications <- &Notification{
						To:     backer.Address.String(),
						Msg:    fmt.Sprintf("A claim you backed has completed. %s", msg),
						TypeID: story.ID,
						Type:   db.NotificationStoryAction,
						Meta:   meta,
					}
				}
				for _, challenger := range story.Challengers {
					if story.Creator.String() == challenger.Address.String() {
						creatorStaked = true
					}

					earns, ok := usersEarns[challenger.Address.String()]
					if !ok {
						s.log.WithField("address", challenger.Address.String()).Info("earns not found")
					}

					msg := getResultMessage(story.StakeDistributionResults.Type, false, challenger, earns)
					notifications <- &Notification{
						To:     challenger.Address.String(),
						Msg:    fmt.Sprintf("A claim you challenged has completed. %s", msg),
						TypeID: story.ID,
						Type:   db.NotificationStoryAction,
						Meta:   meta,
					}
				}
				// if creator also staked he will receive the summary otherwise just a regular notification.
				if creatorStaked {
					continue
				}
				notifications <- &Notification{
					To:     story.Creator.String(),
					Msg:    "A claim you made has completed",
					TypeID: story.ID,
					Type:   db.NotificationStoryAction,
					Meta:   meta,
				}
			}
		}
	}
}

// UserEarns contains users information of stake earned
// when a story completes.
type UserEarns struct {
	Reward   sdk.Coin
	Interest sdk.Coin
}

func processStorySummary(story truchain.CompletedStory) map[string]UserEarns {
	users := make(map[string]UserEarns)
	for _, reward := range story.StakeDistributionResults.Rewards {
		acc := reward.Account.String()
		user, ok := users[acc]
		if !ok {

			user = UserEarns{}
			users[acc] = user
		}
		user.Reward = reward.Amount
		users[acc] = user

	}

	for _, interest := range story.InterestDistributionResults.Interests {
		acc := interest.Account.String()
		user, ok := users[acc]
		if !ok {
			user = UserEarns{}
			users[acc] = user
		}
		user.Interest = interest.Amount
		users[acc] = user
	}

	return users
}
