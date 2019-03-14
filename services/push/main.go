package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	truchain "github.com/TruStory/truchain/types"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
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
			pushEvent := e.(types.EventDataTx)

			var pushData truchain.PushData
			err := json.Unmarshal(pushEvent.Result.Data, &pushData)
			if err != nil {
				panic(err)
			}

			from := pushData.From.String()

			for _, tag := range pushEvent.Result.Tags {
				action := string(tag.Value)

				alert := ""
				switch action {
				case "back_story":
					alert = "%s backed your story"
				case "create_challenge":
					alert = "%s challenged your story"
				}

				if len(alert) > 0 {
					payload := fmt.Sprintf(`{"aps":{"alert":"%s"}}`, fmt.Sprintf(alert, from))

					cert, err := certificate.FromP12File("aps_cert.p12", "trust53]Jean")
					if err != nil {
						log.Fatal("Cert Error:", err)
					}

					notification := &apns2.Notification{}
					notification.DeviceToken = "f329405ac61190bcb23731c39162d5e2650f4a9dea8f357b1cfbc79ddd2990b4"
					notification.Topic = "io.trustory.alpha2"
					notification.Payload = []byte(payload)

					client := apns2.NewClient(cert).Development()
					res, err := client.Push(notification)

					if err != nil {
						log.Fatal("Error:", err)
					}

					fmt.Printf("%v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
				}
			}
		}
	}
}
