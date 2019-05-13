package chttp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
	chain "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/users"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/gorilla/mux"
	"github.com/oklog/ulid"
	"github.com/spf13/viper"
	amino "github.com/tendermint/go-amino"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	tcmn "github.com/tendermint/tendermint/libs/common"
	trpctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tmlibs/cli"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

// MsgTypes is a map of `Msg` type names to empty instances
type MsgTypes map[string]interface{}

// App is implemented by a Cosmos app client to provide chain functionality to the API
type App interface {
	RegisterKey(tcmn.HexBytes, string) (sdk.AccAddress, uint64, sdk.Coins, error)
	RunQuery(string, interface{}) abci.ResponseQuery
	DeliverPresigned(auth.StdTx) (*trpctypes.ResultBroadcastTxCommit, error)
}

// API presents the functionality of a Cosmos app over HTTP
type API struct {
	apiCtx    truCtx.TruAPIContext
	Supported MsgTypes
	router    *mux.Router
}

// NewAPI creates an `API` struct from a client context and a `MsgTypes` schema
func NewAPI(apiCtx truCtx.TruAPIContext, supported MsgTypes) *API {
	a := API{apiCtx: apiCtx, Supported: supported, router: mux.NewRouter()}
	return &a
}

// HandleFunc registers a `chttp.Handler` on the API router
func (a *API) HandleFunc(path string, h Handler) {
	a.router.HandleFunc(path, h.HandlerFunc())
}

// Subrouter returns a mux subrouter.
func (a *API) Subrouter(path string) *mux.Router {
	return a.router.PathPrefix(path).Subrouter()
}

// PathPrefix adds a http.Handler to a path prefix
func (a *API) PathPrefix(path string, handler http.Handler) {
	a.router.PathPrefix(path).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})
}

// Handle registers a http.Handler
func (a *API) Handle(path string, handler http.Handler) {
	a.router.Handle(path, handler)
}

// Use registers a middleware on the API router
func (a *API) Use(mw func(http.Handler) http.Handler) {
	a.router.Use(mw)
}

// ListenAndServe serves HTTP using the API router
func (a *API) ListenAndServe(addr string) error {
	letsEncryptEnabled := a.apiCtx.HTTPSEnabled == true
	fmt.Println("IN HERE 1")
	if !letsEncryptEnabled {
		fmt.Println("IN HERE 2")
		return http.ListenAndServe(addr, a.router)
	}
	fmt.Println("IN HERE 3")
	return a.listenAndServeTLS()
}

func (a *API) listenAndServeTLS() error {
	host := os.Getenv("CHAIN_HOST")
	certDir := os.Getenv("CHAIN_LETS_ENCRYPT_CACHE_DIR")
	if certDir == "" {
		certDir = "certs"
	}
	m := &autocert.Manager{
		Cache:      autocert.DirCache(certDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host),
	}
	httpServer := &http.Server{
		Addr:    ":http",
		Handler: http.HandlerFunc(redirectHandler),
	}
	secureServer := &http.Server{
		Addr:      ":https",
		Handler:   a.router,
		TLSConfig: m.TLSConfig(),
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		return secureServer.ListenAndServeTLS("", "")
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			_ = httpServer.Shutdown(ctx)
			_ = secureServer.Shutdown(ctx)
			return nil
		}
	})

	return g.Wait()
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}

// RegisterKey generates a new address/account for a public key
func (a *API) RegisterKey(k tcmn.HexBytes, algo string) (
	accAddr sdk.AccAddress, accNum uint64, coins sdk.Coins, err error) {

	var addr []byte
	if string(algo[0]) == "*" {
		addr = []byte("cosmostestingaddress")
		algo = algo[1:]
	} else {
		addr = generateAddress()
	}

	tx, err := a.signedRegistrationTx(addr, k, algo)
	if err != nil {
		fmt.Println("TX Parse error: ", err, tx)
		return
	}

	res, err := a.DeliverPresigned(tx)
	if err != nil {
		fmt.Println("TX Broadcast error: ", err, res)
		return
	}

	addresses := users.QueryUsersByAddressesParams{
		Addresses: []string{sdk.AccAddress(addr).String()},
	}
	result, err := a.RunQuery(users.QueryPath, addresses)
	if err != nil {
		return
	}

	var u []users.User
	err = amino.UnmarshalJSON(result, u)
	if err != nil {
		panic(err)
	}
	if len(u) == 0 {
		err = errors.New("Unable to locate account " + string(addr))
		return
	}
	stored := u[0]

	return sdk.AccAddress(addr), stored.AccountNumber, stored.Coins, nil
}

// GenerateAddress returns the first 20 characters of a ULID (https://github.com/oklog/ulid)
func generateAddress() []byte {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	ulidaddr := ulid.MustNew(ulid.Timestamp(t), entropy)
	addr := []byte(ulidaddr.String())[:20]

	return addr
}

func (a *API) signedRegistrationTx(addr []byte, k tcmn.HexBytes, algo string) (auth.StdTx, error) {
	// query registrar account
	// TODO: query using Cosmos auth module (GET account/, {sdk.AccAddress})
	addresses := users.QueryUsersByAddressesParams{
		Addresses: []string{chain.RegistrarAccAddress},
	}
	res, err := a.RunQuery(users.QueryPath, addresses)
	if err != nil {
		return auth.StdTx{}, err
	}

	var u []users.User
	err = amino.UnmarshalJSON(res, u)
	if err != nil {
		panic(err)
	}
	registrarAcc := u[0]

	registrarNum := registrarAcc.AccountNumber
	registrarSequence := registrarAcc.Sequence
	registrationMemo := "reg"
	chainID := "chain-id"
	msg := users.RegisterKeyMsg{
		Address:    addr,
		PubKey:     k,
		PubKeyAlgo: algo,
		Coins:      nil,
	}

	// Sign tx as registrar
	bytesToSign := auth.StdSignBytes(chainID, registrarNum, registrarSequence, chain.RegistrationFee, []sdk.Msg{msg}, registrationMemo)
	registrarKey := loadRegistrarKey()
	sigBytes, err := registrarKey.Sign(bytesToSign)
	if err != nil {
		return auth.StdTx{}, err
	}

	// Construct and submit signed tx
	tx := auth.StdTx{
		Msgs: []sdk.Msg{msg},
		Fee:  chain.RegistrationFee,
		Signatures: []auth.StdSignature{auth.StdSignature{
			PubKey:    registrarKey.PubKey(),
			Signature: sigBytes,
		}},
		Memo: registrationMemo,
	}

	return tx, nil
}

func loadRegistrarKey() secp256k1.PrivKeySecp256k1 {
	rootdir := viper.GetString(cli.HomeFlag)
	if rootdir == "" {
		rootdir = "$HOME/.apid"
	}
	keypath := filepath.Join(rootdir, "registrar.key")
	fileBytes, err := ioutil.ReadFile(keypath)
	if err != nil {
		panic(err)
	}

	keyBytes, err := hex.DecodeString(string(fileBytes))
	if err != nil {
		panic(err)
	}
	if len(keyBytes) != 32 {
		panic("Invalid registrar key: " + string(fileBytes))
	}
	key := secp256k1.PrivKeySecp256k1{}
	copy(key[:], keyBytes)

	return key
}

// RunQuery dispatches a query (path + params) to the Tendermint node
func (a *API) RunQuery(path string, params interface{}) ([]byte, error) {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := a.apiCtx.QueryWithData("/custom/"+path, paramBytes)
	if err != nil {
		return res, err
	}

	return res, nil
}

// DeliverPresigned dispatches a pre-signed transaction to the Tendermint node
func (a *API) DeliverPresigned(tx auth.StdTx) (sdk.TxResponse, error) {
	txBytes := a.apiCtx.Codec.MustMarshalBinaryLengthPrefixed(tx)
	return a.apiCtx.BroadcastTx(txBytes)
}
