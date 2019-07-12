package context

import (
	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
)

// AppConfig is the config for the app
type AppConfig struct {
	Name             string
	URL              string
	MockRegistration bool   `mapstructure:"mock-registration"`
	UploadURL        string `mapstructure:"upload-url"`
	S3AssetsURL      string `mapstructure:"s3-assets-url"`
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
	Name          string
	Port          int
	HTTPSEnabled  bool   `mapstructure:"https-enabled"`
	HTTPSCacheDir string `mapstructure:"https-cache-dir"`
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

// ParamsConfig is the config for the miscellaneous params
type ParamsConfig struct {
	CommentMinLength int   `mapstructure:"comment-min-length"`
	CommentMaxLength int   `mapstructure:"comment-max-length"`
	BlockInterval    int   `mapstructure:"block-interval"`
	DefaultStake     int64 `mapstructure:"default-stake"`
}

// Config contains all the config variables for the API server
type Config struct {
	ChainID   string `mapstructure:"chain-id"`
	App       AppConfig
	Cookie    CookieConfig
	Database  DatabaseConfig
	Flag      FlagConfig
	Host      HostConfig
	Push      PushConfig
	Registrar RegistrarConfig
	Twitter   TwitterConfig
	Web       WebConfig
	Params    ParamsConfig
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
