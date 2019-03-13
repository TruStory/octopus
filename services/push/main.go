package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
)

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
			testE := e.(types.EventDataTx)
			fmt.Println(testE.Result.String())
		}
	}
}
