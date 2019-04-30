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

func getResultMessage(t truchain.StakeDistributionResultsType, isBacker bool, staker truchain.Staker, earns UserEarns) string {
	var resultMsg = ""
	switch t {
	case truchain.DistributionMajorityNotReached:

		resultMsg = fmt.Sprintf("It's a tie but you earned %s interest", getTrustakeValue(earns.Interest))
	case truchain.DistributionBackersWin:
		if isBacker {
			resultMsg = fmt.Sprintf("You won %s TruStake and earned %s interest",
				getTrustakeValue(earns.Reward),
				getTrustakeValue(earns.Interest))
			return resultMsg
		}
		resultMsg = fmt.Sprintf("You lost %s TruStake but earned %s interest",
			getTrustakeValue(staker.Amount),
			getTrustakeValue(earns.Interest),
		)

	case truchain.DistributionChallengersWin:
		if isBacker {
			resultMsg = fmt.Sprintf("You lost %s TruStake but earned %s interest",
				getTrustakeValue(staker.Amount),
				getTrustakeValue(earns.Interest),
			)
			return resultMsg
		}

		resultMsg = fmt.Sprintf("You won %s TruStake and earned %s interest",
			getTrustakeValue(earns.Reward),
			getTrustakeValue(earns.Interest))

	}
	return resultMsg
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
				for _, backer := range story.Backers {
					if story.Creator.String() == backer.Address.String() {
						creatorStaked = true
					}
					earns, ok := usersEarns[backer.Address.String()]

					if !ok {
						fmt.Println("backin earner not found for ", backer.Address.String())
					}

					msg := getResultMessage(story.StakeDistributionResults.Type, true, backer, earns)
					notifications <- &Notification{
						To:     backer.Address.String(),
						Msg:    fmt.Sprintf("A claim you backed has completed. %s", msg),
						TypeID: story.ID,
						Type:   db.NotificationStoryAction,
					}
				}
				for _, challenger := range story.Challengers {
					if story.Creator.String() == challenger.Address.String() {
						creatorStaked = true
					}

					earns, ok := usersEarns[challenger.Address.String()]
					if !ok {
						fmt.Println("challenge earner not found for ", challenger.Address.String())
					}

					msg := getResultMessage(story.StakeDistributionResults.Type, false, challenger, earns)
					notifications <- &Notification{
						To:     challenger.Address.String(),
						Msg:    fmt.Sprintf("A claim you challenged has completed. %s", msg),
						TypeID: story.ID,
						Type:   db.NotificationStoryAction,
					}
				}
				if creatorStaked {
					continue
				}
				notifications <- &Notification{
					To:     story.Creator.String(),
					Msg:    "A claim you made has completed",
					TypeID: story.ID,
					Type:   db.NotificationStoryAction,
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
