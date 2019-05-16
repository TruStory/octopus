package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
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
	flagTwitterOAUTHCallback = "twitter.oauth.callback"
	flagFlagLimit            = "flag.limit"
	flagFlagAdmin            = "flag.admin"
	flagRegistrarName        = "registrar.name"
	flagRegistrarAddr        = "registrar.addr"
	flagRegistrarPass        = "registrar.password"
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
	// TODO: add require trust-node OR chain-id and --home?
	viper.BindPFlag(client.FlagChainID, rootCmd.PersistentFlags().Lookup(client.FlagChainID))

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Failed executing CLI command: %s, exiting...\n", err)
		os.Exit(1)
	}
}

// ./bin/trucli config chain-id test-chain-K8fT26
// ./bin/trucli keys add registrar --home /Users/blockshane
// ./bin/truchaind add-genesis-account $(./bin/trucli keys show registrar -a --home /Users/blockshane) 1000trusteak,1000trustake
// ./bin/truchaind unsafe-reset-all
// ./bin/truchaind start
// ./bin/truapid start --chain-id test-chain-K8fT26

func startCmd(codec *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start API daemon, a local HTTP server",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var config context.Config
			err = viper.Unmarshal(&config)
			if err != nil {
				panic(err)
			}

			// rootDir := viper.GetString(client.HomeFlag)
			// TODO: check if keystore is the same for trucli and truapid
			viper.Set("home", "/Users/blockshane")
			rootDir := viper.GetString("home")
			fmt.Printf("--home flag is %s\n", rootDir)

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

			port := strconv.Itoa(apiCtx.Config.Host.Port)
			log.Fatal(truAPI.ListenAndServe(net.JoinHostPort(apiCtx.Config.Host.Name, port)))

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
	cmd = registerFlagFlags(cmd)
	cmd = registerRegistrarFlags(cmd)

	return cmd
}

func registerAppFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagAppName, "TruStory", "Name of the app")
	viper.BindPFlag(flagAppName, cmd.Flags().Lookup(flagAppName))

	cmd.Flags().String(flagAppURL, "http://localhost:3000", "URL for the app")
	viper.BindPFlag(flagAppURL, cmd.Flags().Lookup(flagAppURL))

	cmd.Flags().Bool(flagAppMockRegistration, true, "Enables mock user registration")
	viper.BindPFlag(flagAppMockRegistration, cmd.Flags().Lookup(flagAppMockRegistration))

	cmd.Flags().String(flagAppUploadURL, "http://ec2-18-144-34-125.us-west-1.compute.amazonaws.com:4000/v1/upload/aws", "S3 upload URL for app media")
	viper.BindPFlag(flagAppUploadURL, cmd.Flags().Lookup(flagAppUploadURL))

	return cmd
}

func registerCookieFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagCookieHashKey, "0f0ee5b192c014069b85802b727b04a9c41d51b67cdc2b498e9ff60f31ad7b7b4cb573c9745eaef2bb242016747f264db427b4387f4d71579e158cdeaefc51b0", "Hash key of cookie")
	viper.BindPFlag(flagCookieHashKey, cmd.Flags().Lookup(flagCookieHashKey))

	cmd.Flags().String(flagCookieEncryptKey, "f9cc632d41202396cfc432cb89ac9eaa8ff3ad96ecd555a378a807954f6c46ec", "Encrypt key of cookie")
	viper.BindPFlag(flagCookieEncryptKey, cmd.Flags().Lookup(flagCookieEncryptKey))

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

func registerFlagFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Int(flagFlagLimit, 4294967295, "Number of flags needed to hide content")
	viper.BindPFlag(flagFlagLimit, cmd.Flags().Lookup(flagFlagLimit))

	cmd.Flags().String(flagFlagAdmin, "cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7", "Flag admin account")
	viper.BindPFlag(flagFlagAdmin, cmd.Flags().Lookup(flagFlagAdmin))

	return cmd
}

func registerRegistrarFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagRegistrarName, "registrar", "Registrar account name")
	viper.BindPFlag(flagRegistrarName, cmd.Flags().Lookup(flagRegistrarName))

	cmd.Flags().String(flagRegistrarAddr, "", "Registrar account address")
	viper.BindPFlag(flagRegistrarAddr, cmd.Flags().Lookup(flagRegistrarAddr))

	cmd.Flags().String(flagRegistrarPass, "", "Registrar account password")
	viper.BindPFlag(flagRegistrarPass, cmd.Flags().Lookup(flagRegistrarPass))

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
		fmt.Printf("Can't read config: %s. Using flags, environment variables, or defaults.\n", err)
	}
}
