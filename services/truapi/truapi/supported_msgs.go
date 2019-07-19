package truapi

import (
	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
	"github.com/TruStory/truchain/x/slashing"
)

var supported = chttp.MsgTypes{
	"MsgCreateClaim":           claim.MsgCreateClaim{},
	"MsgSubmitArgument":        staking.MsgSubmitArgument{},
	"MsgSubmitUpvote":          staking.MsgSubmitUpvote{},
	"MsgEditArgument":          staking.MsgEditArgument{},
	"MsgSlashArgument":         slashing.MsgSlashArgument{},
}
