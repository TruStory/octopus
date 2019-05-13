package cmd

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/TruStory/octopus/services/truapi/context"
	"github.com/TruStory/octopus/services/truapi/truapi"
	chain "github.com/TruStory/truchain/app"
	"github.com/cosmos/cosmos-sdk/client"
	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagAppName = "app.name"
)

var (
	// Used for flags.
	configFile string

	rootCmd = &cobra.Command{
		Use:   "truapi",
		Short: "TruStory API command-line interface",
	}
)

// Execute executes the root command.
func Execute() {
	cobra.OnInitialize(initConfig)
	codec := chain.MakeCodec()
	rootCmd.AddCommand(startCmd(codec))
	rootCmd.PersistentFlags().String(client.FlagChainID, "", "Chain ID of tendermint node")
	// rootCmd.MarkPersistentFlagRequired(client.FlagChainID)
	viper.BindPFlag(client.FlagChainID, rootCmd.PersistentFlags().Lookup(client.FlagChainID))

	// viper.AutomaticEnv()
	// Add flags and prefix all env exposed with TRU
	// 	executor := cli.PrepareMainCmd(rootCmd, "TRU", app.DefaultCLIHome)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Failed executing CLI command: %s, exiting...\n", err)
		os.Exit(1)
	}
}

func startCmd(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start API daemon, a local HTTP server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// fmt.Println(cmd.Flag(client.FlagTrustNode).Value.String())

			var config context.Config
			err = viper.Unmarshal(&config)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%+v\n", config)

			cliCtx := sdkContext.NewCLIContext().WithCodec(codec).WithAccountDecoder(codec)
			apiCtx := context.NewTruAPIContext(&cliCtx, config)
			truAPI := truapi.NewTruAPI(apiCtx)
			truAPI.RegisterMutations()
			truAPI.RegisterOAuthRoutes(apiCtx)
			truAPI.RegisterResolvers()
			truAPI.RegisterRoutes(apiCtx)

			err = truAPI.RunNotificationSender(apiCtx)
			if err != nil {
				fmt.Println("Notification sender could not be started: ", err)
				os.Exit(1)
			}

			log.Fatal(truAPI.ListenAndServe(net.JoinHostPort(apiCtx.Config.Host.Name, apiCtx.Config.Host.Port)))

			return err
		},
	}
	client.RegisterRestServerFlags(cmd)

	// TODO: why doesn't this work?
	cmd.Flags().Bool("https-enabled", false, "Use HTTPS for server")
	viper.BindPFlag("https-enabled", cmd.Flags().Lookup("https-enabled"))

	cmd.Flags().String(flagAppName, "TruStory", "Name of the app")
	viper.BindPFlag(flagAppName, cmd.Flags().Lookup(flagAppName))

	return cmd
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".truapid/config")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Can't read config: %s. Using environment variables or defaults.\n", err)
	}

	fmt.Println(flagAppName + " = " + viper.GetString(flagAppName))
}
