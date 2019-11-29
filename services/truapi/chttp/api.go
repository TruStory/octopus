package chttp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/tendermint/tendermint/crypto/secp256k1"

	app "github.com/TruStory/truchain/types"
	"github.com/TruStory/truchain/x/account"
	"github.com/TruStory/truchain/x/bank"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/gorilla/mux"
	abci "github.com/tendermint/tendermint/abci/types"
	tcmn "github.com/tendermint/tendermint/libs/common"
	trpctypes "github.com/tendermint/tendermint/rpc/core/types"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	truCtx "github.com/TruStory/octopus/services/truapi/context"
)

// MsgTypes is a map of `Msg` type names to empty instances
type MsgTypes map[string]interface{}

// App is implemented by a Cosmos app client to provide chain functionality to the API
type App interface {
	RegisterKey(tcmn.HexBytes, string) (sdk.AccAddress, error)
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

func (a *API) redirectHTTPS() http.Handler {
	if !a.apiCtx.Config.Host.HTTPSRedirect {
		return a.router
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/liveness-probe/status" {
			w.WriteHeader(http.StatusOK)
			return
		}
		forwarded := r.Header.Get("X-Forwarded-Proto")
		if forwarded != "https" {
			url := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
			http.Redirect(w, r, url, http.StatusMovedPermanently)
			return
		}
		a.router.ServeHTTP(w, r)
	})
}

// ListenAndServe serves HTTP using the API router
func (a *API) ListenAndServe(addr string) error {
	letsEncryptEnabled := a.apiCtx.Config.Host.HTTPSEnabled
	if !letsEncryptEnabled {
		return http.ListenAndServe(addr, a.redirectHTTPS())
	}
	return a.listenAndServeTLS()
}

func (a *API) listenAndServeTLS() error {
	m := &autocert.Manager{
		Cache:      autocert.DirCache(a.apiCtx.Config.Host.HTTPSCacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(a.apiCtx.Config.Host.HTTPSDomainWhitelist...),
	}
	httpServer := &http.Server{
		Addr:    ":http",
		Handler: a.redirectHTTPS(),
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
		<-ctx.Done()
		_ = httpServer.Shutdown(ctx)
		_ = secureServer.Shutdown(ctx)
		return nil
	})

	return g.Wait()
}

// RegisterKey generates a new address/account for a public key
func (a *API) RegisterKey(k tcmn.HexBytes, algo string, registrarAccountNumber, registrarSequence uint64) (accAddr sdk.AccAddress, err error) {

	var addr []byte
	if string(algo[0]) == "*" {
		addr = []byte("cosmostestingaddress")
		algo = algo[1:]
	} else {
		addr, err = deriveAddress(k.String())
		if err != nil {
			return
		}
	}

	_, err = a.signAndBroadcastRegistrationTx(addr, k, algo, registrarAccountNumber, registrarSequence)
	if err != nil {
		return
	}

	queryRoute := path.Join(account.QuerierRoute, account.QueryAppAccount)
	res, err := a.Query(queryRoute, account.QueryAppAccountParams{Address: sdk.AccAddress(addr)}, account.ModuleCodec)
	if err != nil {
		return
	}

	var stored = new(account.AppAccount)
	err = account.ModuleCodec.UnmarshalJSON(res, stored)
	if err != nil {
		panic(err)
	}

	return stored.PrimaryAddress(), nil
}

// deriveAddress derives the address from the public key
func deriveAddress(pk string) ([]byte, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, err
	}
	var pkSecp secp256k1.PubKeySecp256k1
	copy(pkSecp[:], pkBytes[:])

	address, err := sdk.AccAddressFromHex(pkSecp.Address().String())
	if err != nil {
		return nil, err
	}

	return address.Bytes(), nil
}

func (a *API) signAndBroadcastRegistrationTx(addr []byte, k tcmn.HexBytes, algo string, registrarAccountNumber, registrarSequence uint64) (res sdk.TxResponse, err error) {
	cliCtx := a.apiCtx
	config := cliCtx.Config.Registrar

	registrarAddr, err := sdk.AccAddressFromBech32(config.Addr)
	if err != nil {
		return
	}
	sk, err := StdKey(algo, k)
	if err != nil {
		return
	}
	msg := account.NewMsgRegisterKey(registrarAddr, addr, sk, algo, sdk.NewCoins(app.InitialStake))
	err = msg.ValidateBasic()
	if err != nil {
		return
	}

	txBldr := auth.NewTxBuilderFromCLI().WithAccountNumber(registrarAccountNumber).WithSequence(registrarSequence).WithTxEncoder(utils.GetTxEncoder(cliCtx.Codec))
	txBytes, err := txBldr.BuildAndSign(config.Name, config.Pass, []sdk.Msg{msg})
	if err != nil {
		return
	}
	fmt.Println("tx -- ", string(txBytes))

	// broadcast to a Tendermint node
	res, err = cliCtx.WithBroadcastMode(client.BroadcastBlock).BroadcastTx(txBytes)
	if err != nil {
		return
	}
	fmt.Println(res)

	return res, nil
}

// SendGiftToAddress sends gift coins to any user
func (a *API) SendGiftToAddress(address string, amount sdk.Coin, brokerAccountNumber, brokerSequence uint64, memo string) error {
	recipient, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return err
	}

	_, err = a.signAndBroadcastGiftTx(recipient, amount, brokerAccountNumber, brokerSequence, memo)
	if err != nil {
		return err
	}

	return nil
}

func (a *API) signAndBroadcastGiftTx(recipient sdk.AccAddress, amount sdk.Coin, brokerAccountNumber, brokerSequence uint64, memo string) (res sdk.TxResponse, err error) {
	cliCtx := a.apiCtx
	config := cliCtx.Config.RewardBroker

	brokerAddr, err := sdk.AccAddressFromBech32(config.Addr)
	if err != nil {
		return
	}

	msg := bank.NewMsgSendGift(brokerAddr, recipient, amount)
	err = msg.ValidateBasic()
	if err != nil {
		fmt.Println(err)
		return
	}

	// build and sign the transaction
	txBldr := auth.NewTxBuilderFromCLI().
		WithAccountNumber(brokerAccountNumber).
		WithSequence(brokerSequence).
		WithTxEncoder(utils.GetTxEncoder(cliCtx.Codec)).
		WithMemo(memo)
	txBytes, err := txBldr.BuildAndSign(config.Name, config.Pass, []sdk.Msg{msg})
	if err != nil {
		fmt.Println(err)
		return
	}

	// broadcast to a Tendermint node
	res, err = cliCtx.WithBroadcastMode(client.BroadcastBlock).BroadcastTx(txBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res)

	return res, nil
}

// RunQuery dispatches a query (path + params) to the Tendermint node
// deprecated: use Amino encoded Query() instead
func (a *API) RunQuery(path string, params interface{}) ([]byte, error) {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	res, _, err := a.apiCtx.QueryWithData("/custom/"+path, paramBytes)
	if err != nil {
		return res, err
	}

	return res, nil
}

// Query dispatches a query to the Tendermint node with Amino encoded params
func (a *API) Query(path string, params interface{}, cdc *codec.Codec) ([]byte, error) {
	paramBytes, err := cdc.MarshalJSON(params)
	if err != nil {
		return nil, err
	}
	res, _, err := a.apiCtx.QueryWithData("/custom/"+path, paramBytes)
	if err != nil {
		return res, err
	}

	return res, nil
}

// DeliverPresigned dispatches a pre-signed transaction to the Tendermint node
func (a *API) DeliverPresigned(tx auth.StdTx) (res sdk.TxResponse, err error) {
	ctx := a.apiCtx

	txBytes := ctx.Codec.MustMarshalBinaryLengthPrefixed(tx)
	res, err = ctx.WithBroadcastMode(client.BroadcastBlock).BroadcastTx(txBytes)
	if err != nil {
		return
	}
	fmt.Println(res)

	return res, nil
}
