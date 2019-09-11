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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tmlibs/cli"
)

const (
	flagAppName                    = "app.name"
	flagAppURL                     = "app.url"
	flagAppMockRegistration        = "app.mock.registration"
	flagAppUploadURL               = "app.upload.url"
	flagAppS3AssetsURL             = "app.s3.assets.url"
	flagCookieHashKey              = "cookie.hash.key"
	flagCookieEncryptKey           = "cookie.encrypt.key"
	flagDatabaseHost               = "database.hostname"
	flagDatabasePort               = "database.port"
	flagDatabaseUser               = "database.username"
	flagDatabasePass               = "database.password"
	flagDatabaseName               = "database.db"
	flagDatabasePool               = "database.pool"
	flagHostName                   = "host.name"
	flagHostPort                   = "host.port"
	flagHostHTTPSEnabled           = "host.https.enabled"
	flagHostHTTPSCacheDir          = "host.https.cache.dir"
	flagPushEndpointURL            = "push.endpoint.url"
	flagWebDirectory               = "web.directory"
	flagWebAuthLoginRedir          = "web.auth.login.redir"
	flagWebAuthLogoutRedir         = "web.auth.logout.redir"
	flagWebAuthDeniedRedir         = "web.auth.denied.redir"
	flagWebAuthNotWhitelistedRedir = "web.auth.not.whitelisted.redir"
	flagTwitterAPIKey              = "twitter.api.key"
	flagTwitterAPISecret           = "twitter.api.secret"
	flagTwitterOAUTHCallback       = "twitter.oauth.callback"
	flagFlagLimit                  = "flag.limit"
	flagFlagAdmin                  = "flag.admin"
	flagRegistrarName              = "registrar.name"
	flagRegistrarAddr              = "registrar.addr"
	flagRegistrarPass              = "registrar.password"
	flagRewardBrokerName           = "rewardbroker.name"
	flagRewardBrokerAddr           = "rewardbroker.addr"
	flagRewardBrokerPass           = "rewardbroker.password"
)

var (
	defaultCLIHome = os.ExpandEnv("$HOME/.octopus")

	rootCmd = &cobra.Command{
		Use:   "truapi",
		Short: "TruStory API command-line interface",
	}
)

// Execute executes the root command.
func Execute() {
	cobra.OnInitialize(initConfig)
	codec := chain.MakeCodec()

	rootCmd.PersistentFlags().String(client.FlagChainID, "", "chain ID of tendermint node")
	err := viper.BindPFlag(client.FlagChainID, rootCmd.PersistentFlags().Lookup(client.FlagChainID))
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().String(cli.HomeFlag, defaultCLIHome, "directory for config and data")
	err = viper.BindPFlag(cli.HomeFlag, rootCmd.PersistentFlags().Lookup(cli.HomeFlag))
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(startCmd(codec))

	err = rootCmd.Execute()
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
			var config context.Config
			err = viper.Unmarshal(&config)
			if err != nil {
				panic(err)
			}

			cliCtx := sdkContext.NewCLIContext().WithCodec(codec)
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
			truAPI.RunLeaderboardScheduler(apiCtx)

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
	cmd = registerRewardBrokerFlags(cmd)

	return cmd
}

func registerAppFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagAppName, "TruStory", "Name of the app")
	err := viper.BindPFlag(flagAppName, cmd.Flags().Lookup(flagAppName))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagAppURL, "http://localhost:3000", "URL for the app")
	err = viper.BindPFlag(flagAppURL, cmd.Flags().Lookup(flagAppURL))
	if err != nil {
		panic(err)
	}

	cmd.Flags().Bool(flagAppMockRegistration, true, "Enables mock user registration")
	err = viper.BindPFlag(flagAppMockRegistration, cmd.Flags().Lookup(flagAppMockRegistration))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagAppUploadURL, "http://ec2-54-183-49-244.us-west-1.compute.amazonaws.com:4000/v1/upload/aws", "S3 upload URL for app media")
	err = viper.BindPFlag(flagAppUploadURL, cmd.Flags().Lookup(flagAppUploadURL))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagAppS3AssetsURL, "https://s3-us-west-1.amazonaws.com/trustory/assets", "S3 assets URL")
	err = viper.BindPFlag(flagAppS3AssetsURL, cmd.Flags().Lookup(flagAppS3AssetsURL))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerCookieFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagCookieHashKey, "0f0ee5b192c014069b85802b727b04a9c41d51b67cdc2b498e9ff60f31ad7b7b4cb573c9745eaef2bb242016747f264db427b4387f4d71579e158cdeaefc51b0", "Hash key of cookie")
	err := viper.BindPFlag(flagCookieHashKey, cmd.Flags().Lookup(flagCookieHashKey))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagCookieEncryptKey, "f9cc632d41202396cfc432cb89ac9eaa8ff3ad96ecd555a378a807954f6c46ec", "Encrypt key of cookie")
	err = viper.BindPFlag(flagCookieEncryptKey, cmd.Flags().Lookup(flagCookieEncryptKey))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerDatabaseFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagDatabaseHost, "0.0.0.0", "Database host name")
	err := viper.BindPFlag(flagDatabaseHost, cmd.Flags().Lookup(flagDatabaseHost))
	if err != nil {
		panic(err)
	}

	cmd.Flags().Int(flagDatabasePort, 5432, "Database port number")
	err = viper.BindPFlag(flagDatabasePort, cmd.Flags().Lookup(flagDatabasePort))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagDatabaseUser, "postgres", "Database username")
	err = viper.BindPFlag(flagDatabaseUser, cmd.Flags().Lookup(flagDatabaseUser))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagDatabasePass, "", "Database password")
	err = viper.BindPFlag(flagDatabasePass, cmd.Flags().Lookup(flagDatabasePass))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagDatabaseName, "trudb", "Database name")
	err = viper.BindPFlag(flagDatabaseName, cmd.Flags().Lookup(flagDatabaseName))
	if err != nil {
		panic(err)
	}

	cmd.Flags().Int(flagDatabasePool, 25, "Database connection pool size")
	err = viper.BindPFlag(flagDatabasePool, cmd.Flags().Lookup(flagDatabasePool))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerHostFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagHostName, "0.0.0.0", "Server host")
	err := viper.BindPFlag(flagHostName, cmd.Flags().Lookup(flagHostName))
	if err != nil {
		panic(err)
	}

	cmd.Flags().Int(flagHostPort, 1337, "Server port")
	err = viper.BindPFlag(flagHostPort, cmd.Flags().Lookup(flagHostPort))
	if err != nil {
		panic(err)
	}

	cmd.Flags().Bool(flagHostHTTPSEnabled, false, "HTTPS enabled")
	err = viper.BindPFlag(flagHostHTTPSEnabled, cmd.Flags().Lookup(flagHostHTTPSEnabled))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagHostHTTPSCacheDir, "./certs", "HTTPS cache directory")
	err = viper.BindPFlag(flagHostHTTPSCacheDir, cmd.Flags().Lookup(flagHostHTTPSCacheDir))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerPushFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagPushEndpointURL, "http://localhost:9001", "Push notification service endpoint")
	err := viper.BindPFlag(flagPushEndpointURL, cmd.Flags().Lookup(flagPushEndpointURL))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerWebFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagWebDirectory, "./webapp", "Web app directory")
	err := viper.BindPFlag(flagWebDirectory, cmd.Flags().Lookup(flagWebDirectory))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagWebAuthLoginRedir, "http://localhost:3000/auth-complete", "Web login redirect URL")
	err = viper.BindPFlag(flagWebAuthLoginRedir, cmd.Flags().Lookup(flagWebAuthLoginRedir))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagWebAuthLogoutRedir, "http://localhost:3000", "Web logout redirect URL")
	err = viper.BindPFlag(flagWebAuthLogoutRedir, cmd.Flags().Lookup(flagWebAuthLogoutRedir))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagWebAuthDeniedRedir, "http://localhost:3000/auth-denied", "Web access denied redirect URL")
	err = viper.BindPFlag(flagWebAuthDeniedRedir, cmd.Flags().Lookup(flagWebAuthDeniedRedir))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagWebAuthNotWhitelistedRedir, "http://localhost:3000/auth-not-whitelisted", "User not whitelisted redirect URL")
	err = viper.BindPFlag(flagWebAuthNotWhitelistedRedir, cmd.Flags().Lookup(flagWebAuthNotWhitelistedRedir))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerTwitterFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagTwitterAPIKey, "", "Twitter API key")
	err := viper.BindPFlag(flagTwitterAPIKey, cmd.Flags().Lookup(flagTwitterAPIKey))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagTwitterAPISecret, "", "Twitter API secret")
	err = viper.BindPFlag(flagTwitterAPISecret, cmd.Flags().Lookup(flagTwitterAPISecret))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagTwitterOAUTHCallback, "http://localhost:1337/auth-twitter-callback", "Twitter OAUTH callback URL")
	err = viper.BindPFlag(flagTwitterOAUTHCallback, cmd.Flags().Lookup(flagTwitterOAUTHCallback))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerFlagFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Int(flagFlagLimit, 4294967295, "Number of flags needed to hide content")
	err := viper.BindPFlag(flagFlagLimit, cmd.Flags().Lookup(flagFlagLimit))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagFlagAdmin, "cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7", "Flag admin account")
	err = viper.BindPFlag(flagFlagAdmin, cmd.Flags().Lookup(flagFlagAdmin))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerRegistrarFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagRegistrarName, "registrar", "Registrar account name")
	err := viper.BindPFlag(flagRegistrarName, cmd.Flags().Lookup(flagRegistrarName))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagRegistrarAddr, "", "Registrar account address")
	err = viper.BindPFlag(flagRegistrarAddr, cmd.Flags().Lookup(flagRegistrarAddr))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagRegistrarPass, "", "Registrar account password")
	err = viper.BindPFlag(flagRegistrarPass, cmd.Flags().Lookup(flagRegistrarPass))
	if err != nil {
		panic(err)
	}

	return cmd
}

func registerRewardBrokerFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(flagRewardBrokerName, "reward_broker", "RewardBroker account name")
	err := viper.BindPFlag(flagRewardBrokerName, cmd.Flags().Lookup(flagRewardBrokerName))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagRewardBrokerAddr, "", "RewardBroker account address")
	err = viper.BindPFlag(flagRewardBrokerAddr, cmd.Flags().Lookup(flagRewardBrokerAddr))
	if err != nil {
		panic(err)
	}

	cmd.Flags().String(flagRewardBrokerPass, "", "RewardBroker account password")
	err = viper.BindPFlag(flagRewardBrokerPass, cmd.Flags().Lookup(flagRewardBrokerPass))
	if err != nil {
		panic(err)
	}

	return cmd
}

func initConfig() {
	home := viper.GetString(cli.HomeFlag)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AddConfigPath(home)
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Can't read config: %s. Using flags, environment variables, or defaults.\n", err)
	}
}
