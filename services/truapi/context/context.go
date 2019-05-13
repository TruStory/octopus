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

// TwitterConfig is the config for Twitter
type TwitterConfig struct {
	APIKey        string `mapstructure:"api-key"`
	APISecret     string `mapstructure:"api-secret"`
	OAUTHCallback string `mapstructure:"oath-callback"`
}

// WebConfig is the config for the web app
type WebConfig struct {
	Directory       string
	AuthLoginRedir  string `mapstructure:"auth-login-redir"`
	AuthLogoutRedir string `mapstructure:"auth-logout-redir"`
	AuthDeniedRedir string `mapstructure:"auth-denied-redir"`
}

// Config contains all the config variables for the API server
type Config struct {
	ChainID  string `mapstructure:"chain-id"`
	App      AppConfig
	Cookie   CookieConfig
	Database DatabaseConfig
	Host     HostConfig
	Push     PushConfig
	Twitter  TwitterConfig
	Web      WebConfig
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
