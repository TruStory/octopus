package context

import (
	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
)

// AppConfig is the config for the app
type AppConfig struct {
	Name                   string
	URL                    string
	MockRegistration       bool   `mapstructure:"mock-registration"`
	WhitelistEnabled       bool   `mapstructure:"whitelist-enabled"`
	UploadURL              string `mapstructure:"upload-url"`
	S3AssetsURL            string `mapstructure:"s3-assets-url"`
	MixpanelToken          string `mapstructure:"mixpanel-token"`
	LiveDebateURL          string `mapstructure:"live-debate-url"`
	SlackWebhook           string `mapstructure:"slack-webhook"`
	RequestTruSlackWebhook string `mapstructure:"request-tru-slack-webhook"`
}

// CookieConfig is the config for the cookie
type CookieConfig struct {
	HashKey    string `mapstructure:"hash-key"`
	EncryptKey string `mapstructure:"encrypt-key"`
}

// DatabaseConfig is the database configuration
type DatabaseConfig struct {
	Host string `mapstructure:"hostname"`
	Port int
	User string `mapstructure:"username"`
	Pass string `mapstructure:"password"`
	Name string `mapstructure:"db"`
	Pool int
}

// FlagConfig is the config for flagging content
type FlagConfig struct {
	Limit int
	Admin string
}

// HostConfig is the config for the server host
type HostConfig struct {
	Name                 string
	Domain               string
	Port                 int
	HTTPSRedirect        bool     `mapstructure:"https-redirect"`
	HTTPSEnabled         bool     `mapstructure:"https-enabled"`
	HTTPSDomainWhitelist []string `mapstructure:"https-domain-whitelist"`
	HTTPSCacheDir        string   `mapstructure:"https-cache-dir"`
}

// PushConfig is the config for push notifications
type PushConfig struct {
	EndpointURL string `mapstructure:"endpoint-url"`
}

// RegistrarConfig is the config for the registrar account that signs in users
type RegistrarConfig struct {
	Name string
	Addr string
	Pass string `mapstructure:"password"`
}

// RewardBrokerConfig is the config for the reward broker account that rewards the users
type RewardBrokerConfig struct {
	Name string
	Addr string
	Pass string `mapstructure:"password"`
}

// TwitterConfig is the config for Twitter
type TwitterConfig struct {
	APIKey        string `mapstructure:"api-key"`
	APISecret     string `mapstructure:"api-secret"`
	OAUTHCallback string `mapstructure:"oauth-callback"`
}

// WebConfig is the config for the web app
type WebConfig struct {
	Directory               string
	DirectoryV2             string `mapstructure:"directory-v2"`
	AuthLoginRedir          string `mapstructure:"auth-login-redir"`
	AuthLogoutRedir         string `mapstructure:"auth-logout-redir"`
	AuthDeniedRedir         string `mapstructure:"auth-denied-redir"`
	AuthNotWhitelistedRedir string `mapstructure:"auth-not-whitelisted-redir"`
}

// CommunityConfig is the config for the community
type CommunityConfig struct {
	InactiveCommunities []string `mapstructure:"inactive-communities"`
	BetaCommunities     []string `mapstructure:"beta-communities"`
}

// ParamsConfig is the config for off-chain params
type ParamsConfig struct {
	CommentMinLength      int `mapstructure:"comment-min-length"`
	CommentMaxLength      int `mapstructure:"comment-max-length"`
	BlockInterval         int `mapstructure:"block-interval"`
	TrendingFeedTimeDecay int `mapstructure:"trending-feed-time-decay"`
}

// AdminConfig is the config for the admin authentication
type AdminConfig struct {
	Username string `mapstructure:"admin-username"`
	Password string `mapstructure:"admin-password"`
}

// AWSConfig is the config for the AWS SDK
type AWSConfig struct {
	Region       string `mapstructure:"aws-region"`
	Sender       string `mapstructure:"aws-ses-sender"`
	AccessKey    string `mapstructure:"aws-access-key"`
	AccessSecret string `mapstructure:"aws-access-secret"`
	S3Region     string `mapstructure:"aws-s3-region"`
	S3Bucket     string `mapstructure:"aws-s3-bucket"`
}

// SpotlightConfig is the config for the Spotlight service
type SpotlightConfig struct {
	URL string `mapstructure:"spotlight-url"`
}

// DripperConfig is the config to send the drip campaigns
type DripperConfig struct {
	Key       string                  `mapstructure:"dripper-api-key"`
	Workflows []DripperWorkflowConfig `mapstructure:"dripper-workflows"`
}

// DripperWorkflowConfig represents a drip campaign's config
type DripperWorkflowConfig struct {
	Name       string   `mapstructure:"name"`
	WorkflowID string   `mapstructure:"workflow-id"`
	EmailID    string   `mapstructure:"email-id"`
	Tags       []string `mapstructure:"tags"`
}

// LeaderboardConfig represents leaderboard configuration
type LeaderboardConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Interval is the interval in minutes for how often update leaderboard metrics
	Interval int `mapstructure:"interval"`
	// TopDisplaying is the number of users to show in the leaderboard
	TopDisplaying int `mapstructure:"top-displaying"`
}

// Metrics represents metrics configuration
type MetricsConfig struct {
	Secret string `mapstructure:"secret"`
}

// DefaultsConfig represents the default values
type DefaultsConfig struct {
	AvatarURL string `mapstructure:"default-avatar-url"`
}

type TwilioConfig struct {
	AccountSID string `mapstructure:"twilio-account-sid"`
	AuthToken  string `mapstructure:"twilio-auth-token"`
	From       string `mapstructure:"twilio-from"`
}

// Config contains all the config variables for the API server
type Config struct {
	ChainID      string `mapstructure:"chain-id"`
	App          AppConfig
	Cookie       CookieConfig
	Database     DatabaseConfig
	Flag         FlagConfig
	Host         HostConfig
	Push         PushConfig
	Registrar    RegistrarConfig
	RewardBroker RewardBrokerConfig
	Twitter      TwitterConfig
	Web          WebConfig
	Community    CommunityConfig
	Params       ParamsConfig
	Admin        AdminConfig
	AWS          AWSConfig
	Spotlight    SpotlightConfig
	Dripper      DripperConfig
	Leaderboard  LeaderboardConfig
	Defaults     DefaultsConfig
	Metrics      MetricsConfig
	Twilio       TwilioConfig
}

// TruAPIContext stores the config for the API and the underlying client context
type TruAPIContext struct {
	*sdkContext.CLIContext
	Config Config
}

// NewTruAPIContext creates a new API context
func NewTruAPIContext(cliCtx *sdkContext.CLIContext, config Config) TruAPIContext {
	return TruAPIContext{
		CLIContext: cliCtx,
		Config:     config,
	}
}
