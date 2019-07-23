package truapi

import (
	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/slashing"
	"github.com/TruStory/truchain/x/staking"
)

var supported = chttp.MsgTypes{
	"MsgCreateClaim":    claim.MsgCreateClaim{},
	"MsgEditClaim":      claim.MsgEditClaim{},
	"MsgSubmitArgument": staking.MsgSubmitArgument{},
	"MsgSubmitUpvote":   staking.MsgSubmitUpvote{},
	"MsgEditArgument":   staking.MsgEditArgument{},
	"MsgSlashArgument":  slashing.MsgSlashArgument{},
}
