package truapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/TruStory/octopus/services/truapi/chttp"
	"github.com/TruStory/octopus/services/truapi/truapi/cookies"
	"github.com/TruStory/truchain/x/claim"
	"github.com/TruStory/truchain/x/staking"
)

// HandleUnsigned takes a `HandleUnsignedRequest` and returns a `HandleUnsignedResponse`
func (ta *TruAPI) HandleUnsigned(r *http.Request) chttp.Response {
	txr := new(chttp.UnsignedRequest)
	jsonBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return chttp.SimpleErrorResponse(500, err)
	}

	err = json.Unmarshal(jsonBytes, txr)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	// Get the authenticated user
	user, err := cookies.GetAuthenticatedUser(ta.APIContext, r)
	if err == http.ErrNoCookie {
		return chttp.SimpleErrorResponse(401, err)
	}
	if err != nil {
		return chttp.SimpleErrorResponse(401, err)
	}

	// Fetch keypair of the user
	keyPair, err := ta.DBClient.KeyPairByUserID(user.ID)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}
	if keyPair == nil {
		// keypair doesn't exist
		return chttp.SimpleErrorResponse(400, errors.New("keypair does not exist on the server"))
	}

	tx, err := ta.NewUnsignedStdTx(*txr, *keyPair)
	if err != nil {
		fmt.Println("Error decoding tx: ", err)
		return chttp.SimpleErrorResponse(400, err)
	}

	res, err := ta.DeliverPresigned(tx)
	if err != nil {
		return chttp.SimpleErrorResponse(400, err)
	}

	resBytes, _ := json.Marshal(res)

	data, err := hex.DecodeString(res.Data)
	if err == nil {
		if txr.MsgTypes[0] == "MsgSubmitArgument" {
			argument := new(staking.Argument)
			err := staking.ModuleCodec.UnmarshalJSON(data, argument)
			if err == nil {
				ta.sendArgumentToSlack(*argument)
			}
		} else if txr.MsgTypes[0] == "MsgCreateClaim" {
			c := new(claim.Claim)
			err = claim.ModuleCodec.UnmarshalJSON(data, c)
			if err == nil {
				ta.sendClaimToSlack(*c)
			}
		}
	}

	return chttp.SimpleResponse(200, resBytes)
}
