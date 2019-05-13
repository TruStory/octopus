package context

import (
	sdkContext "github.com/cosmos/cosmos-sdk/client/context"
	"github.com/spf13/viper"
)

// TruAPIContext stores the config for the API and the underlying client context
type TruAPIContext struct {
	*sdkContext.CLIContext
	HTTPSEnabled bool
}

// NewTruAPIContext creates a new API context
func NewTruAPIContext(cliCtx *sdkContext.CLIContext) TruAPIContext {
	return TruAPIContext{
		CLIContext:   cliCtx,
		HTTPSEnabled: viper.GetBool("https-enabled"),
	}
}
