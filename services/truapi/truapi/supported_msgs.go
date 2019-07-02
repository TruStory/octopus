package truapi

import (
	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/truchain/x/backing"
	"github.com/TruStory/truchain/x/challenge"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
	"github.com/TruStory/truchain/x/story"
	"github.com/TruStory/truchain/x/trubank"
)

var supported = chttp.MsgTypes{
	"SubmitStoryMsg":           story.SubmitStoryMsg{},
	"BackStoryMsg":             backing.BackStoryMsg{},
	"LikeBackingArgumentMsg":   backing.LikeBackingArgumentMsg{},
	"CreateChallengeMsg":       challenge.CreateChallengeMsg{},
	"LikeChallengeArgumentMsg": challenge.LikeChallengeArgumentMsg{},
	"PayRewardMsg":             trubank.PayRewardMsg{},
	"MsgCreateClaim":           claim.MsgCreateClaim{},
	"MsgSubmitArgument":        staking.MsgSubmitArgument{},
	"MsgSubmitUpvote":          staking.MsgSubmitUpvote{},
}
