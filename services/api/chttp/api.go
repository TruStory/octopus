package chttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"math/rand"

	cliContext "github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/gorilla/mux"
	abci "github.com/tendermint/tendermint/abci/types"
	tcmn "github.com/tendermint/tendermint/libs/common"
	trpctypes "github.com/tendermint/tendermint/rpc/core/types"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
	"github.com/oklog/ulid"
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
	cliCtx    cliContext.CLIContext
	Supported MsgTypes
	router    *mux.Router
}

// NewAPI creates an `API` struct from a client context and a `MsgTypes` schema
func NewAPI(cliCtx cliContext.CLIContext, supported MsgTypes) *API {
	a := API{cliCtx: cliCtx, Supported: supported, router: mux.NewRouter()}
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
	letsEncryptEnabled := os.Getenv("CHAIN_LETS_ENCRYPT_ENABLED") == "true"
	if !letsEncryptEnabled {
		return http.ListenAndServe(addr, a.router)
	}
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

// RegisterKey generates a new address/account for a public key
// Implements chttp.App
// func (a *API) RegisterKey(k tcmn.HexBytes, algo string) (sdk.AccAddress, uint64, sdk.Coins, error) {
// 	var addr []byte

// 	if string(algo[0]) == "*" {
// 		addr = []byte("cosmostestingaddress")
// 		algo = algo[1:]
// 	} else {
// 		addr = generateAddress()
// 	}

// 	tx, err := app.signedRegistrationTx(addr, k, algo)

// 	if err != nil {
// 		fmt.Println("TX Parse error: ", err, tx)
// 		return sdk.AccAddress{}, 0, sdk.Coins{}, err
// 	}

// 	res, err := app.DeliverPresigned(tx)

// 	if !res.CheckTx.IsOK() {
// 		fmt.Println("TX Broadcast CheckTx error: ", res.CheckTx.Log)
// 		return sdk.AccAddress{}, 0, sdk.Coins{}, errors.New(res.CheckTx.Log)
// 	}

// 	if !res.DeliverTx.IsOK() {
// 		fmt.Println("TX Broadcast DeliverTx error: ", res.DeliverTx.Log)
// 		return sdk.AccAddress{}, 0, sdk.Coins{}, errors.New(res.DeliverTx.Log)
// 	}

// 	if err != nil {
// 		fmt.Println("TX Broadcast error: ", err, res)
// 		return sdk.AccAddress{}, 0, sdk.Coins{}, err
// 	}

// 	accaddr := sdk.AccAddress(addr)
// 	stored := app.accountKeeper.GetAccount(*(app.blockCtx), accaddr)

// 	if stored == nil {
// 		return sdk.AccAddress{}, 0, sdk.Coins{}, errors.New("Unable to locate account " + string(addr))
// 	}

// 	coins := stored.GetCoins()

// 	return accaddr, stored.GetAccountNumber(), coins, nil
// }

// GenerateAddress returns the first 20 characters of a ULID (https://github.com/oklog/ulid)
func generateAddress() []byte {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	ulidaddr := ulid.MustNew(ulid.Timestamp(t), entropy)
	addr := []byte(ulidaddr.String())[:20]

	return addr
}

func (a *API) signedRegistrationTx(addr []byte, k tcmn.HexBytes, algo string) (auth.StdTx, error) {
	msg := users.RegisterKeyMsg{
		Address:    addr,
		PubKey:     k,
		PubKeyAlgo: algo,
		// Coins:      app.initialCoins(),
	}
	chainID := app.blockHeader.ChainID
	registrarAcc := app.accountKeeper.GetAccount(*(app.blockCtx), []byte(types.RegistrarAccAddress))
	registrarNum := registrarAcc.GetAccountNumber()
	registrarSequence := registrarAcc.GetSequence()
	registrationMemo := "reg"

	// Sign tx as registrar
	bytesToSign := auth.StdSignBytes(chainID, registrarNum, registrarSequence, types.RegistrationFee, []sdk.Msg{msg}, registrationMemo)
	sigBytes, err := app.registrarKey.Sign(bytesToSign)

	if err != nil {
		return auth.StdTx{}, err
	}

	// Construct and submit signed tx
	tx := auth.StdTx{
		Msgs: []sdk.Msg{msg},
		Fee:  types.RegistrationFee,
		Signatures: []auth.StdSignature{auth.StdSignature{
			PubKey:    app.registrarKey.PubKey(),
			Signature: sigBytes,
		}},
		Memo: registrationMemo,
	}

	return tx, nil
}

// func (app *TruChain) initialCoins() sdk.Coins {
// 	coins := sdk.Coins{}
// 	categories, err := app.categoryKeeper.GetAllCategories(*(app.blockCtx))
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, cat := range categories {
// 		coin := sdk.NewCoin(cat.Denom(), types.InitialCredAmount)
// 		coins = append(coins, coin)
// 	}

// 	coins = append(coins, types.InitialTruStake)

// 	// coins need to be sorted by denom to be valid
// 	coins.Sort()

// 	// yes we should panic if coins aren't valid
// 	// as it undermines the whole chain
// 	if !coins.IsValid() {
// 		panic("Initial coins are not valid.")
// 	}

// 	return coins
// }

// RunQuery dispatches a query (path + params) to the Tendermint node
func (a *API) RunQuery(path string, params interface{}) ([]byte, error) {
	paramBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	res, err := a.cliCtx.QueryWithData("/custom/"+path, paramBytes)
	if err != nil {
		return res, err
	}

	return res, nil
}

// DeliverPresigned dispatches a pre-signed transaction to the Tendermint node
func (a *API) DeliverPresigned(tx auth.StdTx) (sdk.TxResponse, error) {
	txBytes := a.cliCtx.Codec.MustMarshalBinaryLengthPrefixed(tx)
	return a.cliCtx.BroadcastTx(txBytes)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
