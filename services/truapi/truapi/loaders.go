//go:generate  go run github.com/vektah/dataloaden AppAccountLoader string *github.com/TruStory/octopus/services/truapi/truapi.AppAccount
//go:generate  go run github.com/vektah/dataloaden UserProfileLoader string *github.com/TruStory/octopus/services/truapi/db.UserProfile
package truapi

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ta *TruAPI) AppAccountLoader() *AppAccountLoader {
	config := AppAccountLoaderConfig{
		Fetch: func(keys []string) (accounts []*AppAccount, errors []error) {
			addresses := make([]sdk.AccAddress, 0, len(keys))
			for _, k := range keys {
				address, err := sdk.AccAddressFromBech32(k)
				if err != nil {
					errors = append(errors, err)
					return
				}
				addresses = append(addresses, address)
			}
			accounts, err := ta.appAccountsResolver(context.Background(), addresses)
			if err != nil {
				errors = append(errors, err)
			}
			return
		},
		Wait:     time.Millisecond * 2,
		MaxBatch: 50,
	}
	return NewAppAccountLoader(config)
}
