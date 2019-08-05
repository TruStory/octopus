package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/russross/blackfriday/v2"

	"github.com/TruStory/octopus/services/truapi/postman"
)

type User struct {
	Email string
}

var users []User = []User{
	User{"mohit.mamoria@gmail.com"},
	User{"mamoria.mohit@gmail.com"},
}

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		log.Fatal("please pass the email address from which the emails must be sent. eg. go run *.go preethi@trustory.io")
		os.Exit(1)
	}

	client, err := postman.NewVanillaPostman("us-west-2", args[0])
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for _, user := range users {
		fmt.Printf("Sending email to... %s", user.Email)
		vars := struct {
			SignupLink string
		}{
			SignupLink: "https://beta.trustory.io/signup",
		}

		var body bytes.Buffer
		if err = client.Messages["signup"].Execute(&body, vars); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		message := postman.Message{
			To:      []string{user.Email},
			Subject: "Getting you started with TruStory Beta",
			Body:    string(blackfriday.Run(body.Bytes())),
		}

		if err = client.Deliver(message); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Printf(" ✅\n")
	}
}
