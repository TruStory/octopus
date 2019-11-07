package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

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

	replacers, err := os.Create("replacers.txt")
	if err != nil {
		log.Fatalln(err)
	}

	var users []db.User
	err = dbClient.FindAll(&users)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Found %d users.", len(users))
	for _, user := range users {
		fmt.Printf("\n\nFixing %s with ID: %d (%s)", user.Username, user.ID, user.FullName)
		keyPair, err := dbClient.KeyPairByUserID(user.ID)
		if err != nil {
			log.Fatalln(err)
		}
		if keyPair == nil {
			fmt.Printf("\nDoes not have a key pair yet. Skipping.")
			continue
		}
		fmt.Printf("\nCurrent keys:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, keyPair.PublicKey, user.Address)

		newPubK, newAddress, err := calculatePublicKeyAndAddress(keyPair.PrivateKey)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("\nUpdating them to:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, newPubK, newAddress)
		_, err = fmt.Fprintln(replacers, fmt.Sprintf("%s:%s\n%s:%s", keyPair.PublicKey, newPubK, user.Address, newAddress))
		if err != nil {
			log.Fatalln(err)
		}
	}
	err = replacers.Close()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("\nFinished writing replacers.")
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
