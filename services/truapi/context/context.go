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

// Config contains all the config variables for the API server
type Config struct {
	ChainID      string `mapstructure:"chain-id"`
	HTTPSEnabled bool   `mapstructure:"https-enabled"`
	Database     DatabaseConfig
	Host         string
	Port         string
}

// TruAPIContext stores the config for the API and the underlying client context
type TruAPIContext struct {
	*sdkContext.CLIContext
	HTTPSEnabled bool
	Host         string
	Port         string
}

// NewTruAPIContext creates a new API context
func NewTruAPIContext(cliCtx *sdkContext.CLIContext, config Config) TruAPIContext {
	return TruAPIContext{
		CLIContext:   cliCtx,
		HTTPSEnabled: config.HTTPSEnabled,
		Host:         config.Host,
		Port:         config.Port,
	}
}
