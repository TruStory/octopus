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

// Execute executes the root command.
func Execute() {
	// Add --chain-id to persistent flags and mark it required
	rootCmd.PersistentFlags().String(client.FlagChainID, "", "Chain ID of tendermint node")

	codec := chain.MakeCodec()
	rootCmd.AddCommand(httpCmd(codec))

	// Add flags and prefix all env exposed with TRU
	// 	executor := cli.PrepareMainCmd(rootCmd, "TRU", app.DefaultCLIHome)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Failed executing CLI command: %s, exiting...\n", err)
		os.Exit(1)
	}
}

func httpCmd(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start API daemon, a local HTTP server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
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
	client.RegisterRestServerFlags(cmd)

	return cmd
}
