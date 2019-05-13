package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

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
	flagAppName              = "app.name"
	flagAppURL               = "app.url"
	flagAppMockRegistration  = "app.mock.registration"
	flagAppUploadURL         = "app.upload.url"
	flagCookieHashKey        = "cookie.hash.key"
	flagCookieEncryptKey     = "cookie.encrypt.key"
	flagDatabaseHost         = "database.hostname"
	flagDatabasePort         = "database.port"
	flagDatabaseUser         = "database.username"
	flagDatabasePass         = "database.password"
	flagDatabaseName         = "database.db"
	flagDatabasePool         = "database.pool"
	flagHostName             = "host.name"
	flagHostPort             = "host.port"
	flagHostHTTPSEnabled     = "host.https.enabled"
	flagHostHTTPSCacheDir    = "host.https.cache.dir"
	flagPushEndpointURL      = "push.endpoint.url"
	flagWebDirectory         = "web.directory"
	flagWebAuthLoginRedir    = "web.auth.login.redir"
	flagWebAuthLogoutRedir   = "web.auth.logout.redir"
	flagWebAuthDeniedRedir   = "web.auth.denied.redir"
	flagTwitterAPIKey        = "twitter.api.key"
	flagTwitterAPISecret     = "twitter.api.secret"
	flagTwitterOAUTHCallback = "twitter.oath.callback"
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

	cmd = registerAppFlags(cmd)
	cmd = registerCookieFlags(cmd)
	cmd = registerDatabaseFlags(cmd)
	cmd = registerHostFlags(cmd)
	cmd = registerPushFlags(cmd)
	cmd = registerWebFlags(cmd)
	cmd = registerTwitterFlags(cmd)

	// Config vars can be set in 3 ways:
	// i.e: for the paramter "app.name":
	// 1. Command-line flag: --app.name TruStory
	// 2. Env var: APP_NAME=TruStory
	// 3. config.toml in .truchapid/config ([app] name = TruStory)
	// 4. Default value "TruStory" if not supplied by the above
	// Precedence: 1 -> 2 -> 3 -> 4

	cmd.Flags().String(flagAppName, "TruStory", "Name of the app")
	viper.BindPFlag(flagAppName, cmd.Flags().Lookup(flagAppName))

	cmd.Flags().String(flagAppURL, "http://localhost:3000", "URL for the app")
	viper.BindPFlag(flagAppURL, cmd.Flags().Lookup(flagAppURL))

	return cmd
}

func registerDatabaseFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagDatabaseHost, "0.0.0.0", "Database host name")
	viper.BindPFlag(flagDatabaseHost, cmd.Flags().Lookup(flagDatabaseHost))

	cmd.Flags().Int(flagDatabasePort, 5432, "Database port number")
	viper.BindPFlag(flagDatabasePort, cmd.Flags().Lookup(flagDatabasePort))

	cmd.Flags().String(flagDatabaseUser, "postgres", "Database username")
	viper.BindPFlag(flagDatabaseUser, cmd.Flags().Lookup(flagDatabaseUser))

	cmd.Flags().String(flagDatabasePass, "", "Database password")
	viper.BindPFlag(flagDatabasePass, cmd.Flags().Lookup(flagDatabasePass))

	cmd.Flags().String(flagDatabaseName, "trudb", "Database name")
	viper.BindPFlag(flagDatabaseName, cmd.Flags().Lookup(flagDatabaseName))

	cmd.Flags().Int(flagDatabasePool, 25, "Database connection pool size")
	viper.BindPFlag(flagDatabasePool, cmd.Flags().Lookup(flagDatabasePool))

	return cmd
}

func registerHostFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagHostName, "0.0.0.0", "Server host")
	viper.BindPFlag(flagHostName, cmd.Flags().Lookup(flagHostName))

	cmd.Flags().Int(flagHostPort, 1337, "Server port")
	viper.BindPFlag(flagHostPort, cmd.Flags().Lookup(flagHostPort))

	cmd.Flags().Bool(flagHostHTTPSEnabled, false, "HTTPS enabled")
	viper.BindPFlag(flagHostHTTPSEnabled, cmd.Flags().Lookup(flagHostHTTPSEnabled))

	cmd.Flags().String(flagHostHTTPSCacheDir, "./certs", "HTTPS cache directory")
	viper.BindPFlag(flagHostHTTPSCacheDir, cmd.Flags().Lookup(flagHostHTTPSCacheDir))

	return cmd
}

func registerPushFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagPushEndpointURL, "http://localhost:9001", "Push notification service endpoint")
	viper.BindPFlag(flagPushEndpointURL, cmd.Flags().Lookup(flagPushEndpointURL))

	return cmd
}

func registerWebFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagWebDirectory, "./webapp", "Web app directory")
	viper.BindPFlag(flagWebDirectory, cmd.Flags().Lookup(flagWebDirectory))

	cmd.Flags().String(flagWebAuthLoginRedir, "http://localhost:3000/auth-complete", "Web login redirect URL")
	viper.BindPFlag(flagWebAuthLoginRedir, cmd.Flags().Lookup(flagWebAuthLoginRedir))

	cmd.Flags().String(flagWebAuthLogoutRedir, "http://localhost:3000", "Web logout redirect URL")
	viper.BindPFlag(flagWebAuthLogoutRedir, cmd.Flags().Lookup(flagWebAuthLogoutRedir))

	cmd.Flags().String(flagWebAuthDeniedRedir, "http://localhost:3000/auth-denied", "Web access denied redirect URL")
	viper.BindPFlag(flagWebAuthDeniedRedir, cmd.Flags().Lookup(flagWebAuthDeniedRedir))

	return cmd
}

func registerTwitterFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagTwitterAPIKey, "", "Twitter API key")
	viper.BindPFlag(flagTwitterAPIKey, cmd.Flags().Lookup(flagTwitterAPIKey))

	cmd.Flags().String(flagTwitterAPISecret, "", "Twitter API secret")
	viper.BindPFlag(flagTwitterAPISecret, cmd.Flags().Lookup(flagTwitterAPISecret))

	cmd.Flags().String(flagTwitterOAUTHCallback, "http://localhost:1337/auth-twitter-callback", "Twitter OAUTH callback URL")
	viper.BindPFlag(flagTwitterOAUTHCallback, cmd.Flags().Lookup(flagTwitterOAUTHCallback))

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

		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AddConfigPath(home)
		viper.SetConfigName(".truapid/config")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Can't read config: %s. Using environment variables or defaults.\n", err)
	}

	fmt.Println(flagAppName + " = " + viper.GetString(flagAppName))
}
