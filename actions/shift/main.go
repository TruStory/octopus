package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/TruStory/octopus/actions/shift/shifters"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

const (
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = "tru"
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = "trupub"
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = "truvaloper"
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = "truvaloperpub"
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = "truvalcons"
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = "truvalconspub"
)

var registry = make(map[string]shifters.Shifter)

func init() {
	registry["mixer"] = shifters.MixerShifter{}
	registry["mixpanel"] = shifters.MixpanelShifter{Token: getEnv("MIXPANEL_PROJECT_TOKEN", "")}
}

func main() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.Seal()

	shiftersToRun := os.Args[1:]

	dbPort, err := strconv.Atoi(getEnv("PG_PORT", "5432"))
	if err != nil {
		log.Fatalln(err)
	}
	truConfig := truCtx.Config{
		Database: truCtx.DatabaseConfig{
			Host: getEnv("PG_HOST", "localhost"),
			Port: dbPort,
			User: getEnv("PG_USER", "postgres"),
			Pass: getEnv("PG_USER_PW", ""),
			Name: getEnv("PG_DB_NAME", "trudb"),
			Pool: 25,
		},
	}
	dbClient := db.NewDBClient(truConfig)

	r, err := makeReplacers(dbClient)
	if err != nil {
		log.Fatalln(err)
	}

	for _, s := range shiftersToRun {
		fmt.Printf("\n\n=> Running shifter: %s", s)

		shifter, ok := registry[s]
		if !ok {
			log.Fatal(errors.New("no such shifter found in the registry"))
		}

		err = shifter.Shift(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n=> Completed running shifter: %s\n", s)
	}

	fmt.Printf("\nFinished writing replacers.")
}

func makeReplacers(dbClient *db.Client) (shifters.Replacers, error) {
	var users []db.User
	err := dbClient.FindAll(&users)
	if err != nil {
		return shifters.Replacers{}, err
	}
	fmt.Printf("Found %d users.", len(users))

	r := shifters.Replacers{}
	for _, user := range users {
		fmt.Printf("\n\nMaking replacer for %s with ID: %d (%s)", user.Username, user.ID, user.FullName)
		keyPair, err := dbClient.KeyPairByUserID(user.ID)
		if err != nil {
			return shifters.Replacers{}, err
		}
		if keyPair == nil {
			fmt.Printf("\nDoes not have a key pair yet. Skipping.")
			continue
		}
		fmt.Printf("\nCurrent keys:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, keyPair.PublicKey, user.Address)

		newPubK, newAddress, err := calculatePublicKeyAndAddress(keyPair.PrivateKey)
		if err != nil {
			return shifters.Replacers{}, err
		}
		fmt.Printf("\nWill be changed to:\n\tPriv: %s\n\tPub: %s\n\tAddr: %s", keyPair.PrivateKey, newPubK, newAddress)
		r = append(r, shifters.Replacer{From: keyPair.PublicKey, To: newPubK})
		r = append(r, shifters.Replacer{From: user.Address, To: newAddress})
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
