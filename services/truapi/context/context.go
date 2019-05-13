package context

import (
	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
)

// DatabaseConfig is the database configuration
type DatabaseConfig struct {
	Host string `mapstructure:"hostname"`
	Port string
	User string `mapstructure:"username"`
	Pass string `mapstructure:"password"`
}

// HostConfig is the config for the server host
type HostConfig struct {
	Name          string
	Port          string
	HTTPSEnabled  bool   `mapstructure:"https-enabled"`
	HTTPSCacheDir string `mapstructure:"https-cache-dir"`
}

// Config contains all the config variables for the API server
type Config struct {
	ChainID  string `mapstructure:"chain-id"`
	Host     HostConfig
	Database DatabaseConfig
}

// TruAPIContext stores the config for the API and the underlying client context
type TruAPIContext struct {
	*sdkContext.CLIContext

	ChainID       string
	Host          string
	Port          string
	HTTPSEnabled  bool
	HTTPSCacheDir string
}

// NewTruAPIContext creates a new API context
func NewTruAPIContext(cliCtx *sdkContext.CLIContext, config Config) TruAPIContext {
	return TruAPIContext{
		CLIContext:    cliCtx,
		ChainID:       config.ChainID,
		Host:          config.Host.Name,
		Port:          config.Host.Port,
		HTTPSEnabled:  config.Host.HTTPSEnabled,
		HTTPSCacheDir: config.Host.HTTPSCacheDir,
	}
}
