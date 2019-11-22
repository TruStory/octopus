package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/TruStory/octopus/actions/shift/shifters"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/libs/bech32"
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
	r = append(r, convertUntrackecAddresses()...)
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

var untrackedAddreses = []string{
	"cosmos18rsxqvckda8945hvsupcu99fu7dw3ke0kwf3e0",
	"cosmos1w3e82cmgv95kuctrvdex2emfwd68yctjpzp3mr",
	"cosmos1em44grl9ylmmnwawwp5fjn079kesatwp67rxjx",
	"cosmos1zsfyml5c43ekeq60hm97acklr007tuzerqvw52",
	"cosmos1f7x5wx3adh6klcurmd8n36etx4elgu9d4wkys3",
	"cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh",
	"cosmos1tygms3xhhs3yv487phx3dw4a95jn7t7lpm470r",
	"cosmos1tfpcnjzkthft3ynewqvn7mtdk7guf3knjdqg4d",
	"cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
	"cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl",
	"cosmos1ed82m7snyk8mux8xxpwygvtyq633a4k43rfp8l",
	"cosmos1m3h30wlvsf8llruxtpukdvsy0km2kum8g38c8q",
	"cosmos17xpfvakm2amg962yls6f84z3kell8c5lserqta",
	"cosmosvaloper1tfpcnjzkthft3ynewqvn7mtdk7guf3knhe5ae7",
	"cosmosvalcons14dmnmnzsxc95g822n7a2kd6r88j2ahs66k6rsj",
	"cosmosvalcons1llfh9se57f6a8scv5slecfsdja3q2kvh4dhdu6",
	"cosmosvalconspub1zcjduepqf5hfmgmcsm8quaqfv00yt5s2a6t8ejdj4rsfhjv0d803lpg3zxws9vd47y",
}

func convertUntrackecAddresses() shifters.Replacers {
	r := shifters.Replacers{}
	for _, s := range untrackedAddreses {
		if strings.HasPrefix(s, "cosmosvalconspub") {
			b, err := sdk.GetFromBech32(s, sdk.Bech32PrefixConsPub)
			if err != nil {
				log.Fatal(err, s)
			}
			_, err = sdk.AccAddressFromHex(hex.EncodeToString(b))
			if err != nil {
				log.Fatal(err)
			}

			to, err := bech32.ConvertAndEncode(Bech32PrefixConsPub, b)
			if err != nil {
				log.Fatal(err)
			}
			r = append(r, shifters.Replacer{From: s, To: to})
			continue
		}
		if strings.HasPrefix(s, "cosmosvalcons") {
			b, err := sdk.GetFromBech32(s, sdk.Bech32PrefixConsAddr)
			if err != nil {
				log.Fatal(err)
			}

			_, err = sdk.AccAddressFromHex(hex.EncodeToString(b))
			if err != nil {
				log.Fatal(err)
			}

			to, err := bech32.ConvertAndEncode(Bech32PrefixConsAddr, b)
			if err != nil {
				log.Fatal(err)
			}
			r = append(r, shifters.Replacer{From: s, To: to})
			continue
		}
		if strings.HasPrefix(s, "cosmosvaloperpub") {
			b, err := sdk.GetFromBech32(s, sdk.Bech32PrefixValPub)
			if err != nil {
				log.Fatal(err)
			}

			_, err = sdk.AccAddressFromHex(hex.EncodeToString(b))
			if err != nil {
				log.Fatal(err)
			}

			to, err := bech32.ConvertAndEncode(Bech32PrefixValPub, b)
			if err != nil {
				log.Fatal(err)
			}
			r = append(r, shifters.Replacer{From: s, To: to})
			continue
		}
		if strings.HasPrefix(s, "cosmosvaloper") {
			b, err := sdk.GetFromBech32(s, sdk.Bech32PrefixValAddr)
			if err != nil {
				log.Fatal(err)
			}

			_, err = sdk.AccAddressFromHex(hex.EncodeToString(b))
			if err != nil {
				log.Fatal(err)
			}

			to, err := bech32.ConvertAndEncode(Bech32PrefixValAddr, b)
			if err != nil {
				log.Fatal(err)
			}
			r = append(r, shifters.Replacer{From: s, To: to})
			continue
		}
		if strings.HasPrefix(s, "cosmos") {
			b, err := sdk.GetFromBech32(s, sdk.Bech32PrefixAccAddr)
			if err != nil {
				log.Fatal(err)
			}

			_, err = sdk.AccAddressFromHex(hex.EncodeToString(b))
			if err != nil {
				log.Fatal(err)
			}

			to, err := bech32.ConvertAndEncode(Bech32PrefixAccAddr, b)
			if err != nil {
				log.Fatal(err)
			}
			r = append(r, shifters.Replacer{From: s, To: to})
			continue
		}
	}
	if len(r) != len(untrackedAddreses) {
		log.Fatal("unable to convert all untracked addresses")
	}
	return r
}
