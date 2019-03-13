package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	truchain "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/backing"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

// var msg backing.BackStoryMsg.V

func main() {
	client := client.NewHTTP("tcp://0.0.0.0:26657", "/websocket")
	err := client.Start()
	if err != nil {
		// handle error
	}
	defer client.Stop()
	timeout := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// query := query.MustParse("tm.event='NewBlock'")
	query := query.MustParse("tru.event = 'Push'")
	txs := make(chan interface{})
	err = client.Subscribe(ctx, "trustory-push-client", query, txs)

	// go func() {
	// 	for e := range txs {
	// 		fmt.Println("got ", e.(types.EventDataTx))
	// 	}
	// }()

	for {
		for e := range txs {
			// fmt.Println(e)
			pushEvent := e.(types.EventDataTx)
			fmt.Println(pushEvent.Result.String())

			var pushData truchain.PushData
			err := json.Unmarshal(pushEvent.Result.Data, &pushData)
			if err != nil {
				panic(err)
			}

			fmt.Println(pushData.From.String())

			for _, tag := range pushEvent.Result.Tags {
				fmt.Println(string(tag.Value))
				fmt.Println(backing.BackStoryMsg.Type())
			}

		}
	}
}
