package cmd

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/TruStory/octopus/services/truapi/truapi"
	chain "github.com/TruStory/truchain/app"
	"github.com/TruStory/truchain/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	// cfgFile, userLicense string

	rootCmd = &cobra.Command{
		Use:   "truapi",
		Short: "TruStory API command-line interface",
	}
)

// config
// CHAIN_LETS_ENCRYPT_ENABLED
// CHAIN_HOST
// CHAIN_LETS_ENCRYPT_CACHE_DIR
// home directory -- replace $HOME.apid
// -- registrar.key
// PG_ADDR
// PG_USER
// PG_USER_PW
// PG_DB_NAME
// COOKIE_HASH_KEY
// COOKIE_ENCRYPT_KEY
// APP_NAME
// APP_URL
// AUTH_LOGIN_REDIR
// AUTH_LOGOUT_REDIR
// AUTH_DENIED_REDIR
// TWITTER_API_KEY
// TWITTER_API_SECRET
// CHAIN_OAUTH_CALLBACK
// UPLOAD_URL
// PUSHD_ENDPOINT_URL
// MOCK_REGISTRATION
// CHAIN_WEB_DIR
// types.Hostname -- 0.0.0.0:1337
// types.Portname -- 0.0.0.0:1337

// Execute executes the root command.
func Execute() {
	codec := chain.MakeCodec()
	rootCmd.AddCommand(serverCmd(codec))
	rootCmd.PersistentFlags().String(client.FlagChainID, "", "Chain ID of tendermint node")
	rootCmd.MarkPersistentFlagRequired(client.FlagChainID)

	// Add flags and prefix all env exposed with TRU
	// 	executor := cli.PrepareMainCmd(rootCmd, "TRU", app.DefaultCLIHome)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Failed executing CLI command: %s, exiting...\n", err)
		os.Exit(1)
	}
}

func serverCmd(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start API daemon, a local HTTP server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fmt.Println(cmd.Flag(client.FlagChainID).Value.String())
			// fmt.Println(cmd.Flag(client.FlagTrustNode).Value.String())
			cliCtx := context.NewCLIContext().WithCodec(codec).WithAccountDecoder(codec)
			truAPI := truapi.NewTruAPI(cliCtx)
			truAPI.RegisterMutations()
			truAPI.RegisterOAuthRoutes()
			truAPI.RegisterResolvers()
			truAPI.RegisterRoutes()

			log.Fatal(truAPI.ListenAndServe(net.JoinHostPort(types.Hostname, types.Portname)))

			return err
		},
	}
	// client.RegisterRestServerFlags(cmd)

	return cmd
}
