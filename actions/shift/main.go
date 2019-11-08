package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type replacer struct {
	from string
	to   string
}

type replacers []replacer

func main() {
	dbPort, err := strconv.Atoi(getEnv("PG_PORT", "5432"))
	if err != nil {
		log.Fatalln(err)
	}
	config := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_HOST", "localhost"),
			Port: dbPort,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	dbClient := db.NewDBClient(config)

	r, err := makeReplacers(dbClient)
	if err != nil {
		log.Fatalln(err)
	}

	err = filepath.Walk("mixer", replace(r))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("\nFinished writing replacers.")
}

func replace(r replacers) func(path string, info os.FileInfo, err error) error {
	return func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		fmt.Printf("\nReplacing in file: %s ", path)

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		for _, replacer := range r {
			fmt.Print(".")
			content = bytes.ReplaceAll(content, []byte(replacer.from), []byte(replacer.to))
		}

		fmt.Print("DONE.")

		err = ioutil.WriteFile(path, []byte(content), 0)
		if err != nil {
			return err
		}

		return nil
	}
}

func makeReplacers(dbClient *db.Client) (replacers, error) {
	var users []db.User
	err := dbClient.FindAll(&users)
	if err != nil {
		return replacers{}, err
	}
	fmt.Printf("Found %d users.", len(users))

	r := replacers{}
	for _, user := range users {
		fmt.Printf("\n\nMaking replacer for %s with ID: %d (%s)", user.Username, user.ID, user.FullName)
		keyPair, err := dbClient.KeyPairByUserID(user.ID)
		if err != nil {
			return replacers{}, err
		}
		if keyPair == nil {
			fmt.Printf("\nDoes not have a key pair yet. Skipping.")
			continue
		}
		fmt.Printf("\nCurrent keys:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, keyPair.PublicKey, user.Address)

		newPubK, newAddress, err := calculatePublicKeyAndAddress(keyPair.PrivateKey)
		if err != nil {
			return replacers{}, err
		}
		fmt.Printf("\nWill be changed to:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, newPubK, newAddress)
		r = append(r, replacer{from: keyPair.PublicKey, to: newPubK})
		r = append(r, replacer{from: user.Address, to: newAddress})
	}

	return r, nil
}

func calculatePublicKeyAndAddress(privateKey string) (publicKey string, address string, err error) {
	pk := getPrivateKeyObject(privateKey)

	pubk := hex.EncodeToString(pk.PubKey().SerializeCompressed())

	addr := deriveAddress(pubk)

	return pubk, addr, nil
}

func deriveAddress(publicKey string) string {
	pkBytes, _ := hex.DecodeString(publicKey)
	var pkSecp secp256k1.PubKeySecp256k1
	copy(pkSecp[:], pkBytes[:])

	address, _ := sdk.AccAddressFromHex(pkSecp.Address().String())

	return address.String()
}

func getPrivateKeyObject(privateKey string) *btcec.PrivateKey {
	pkBytes, _ := hex.DecodeString(privateKey)
	privKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), pkBytes)

	return privKey
}
