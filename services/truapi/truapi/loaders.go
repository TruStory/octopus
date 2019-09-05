//go:generate  go run github.com/vektah/dataloaden AppAccountLoader string *github.com/TruStory/octopus/services/truapi/truapi.AppAccount
//go:generate  go run github.com/vektah/dataloaden UserProfileLoader string *github.com/TruStory/octopus/services/truapi/db.UserProfile
package truapi

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/TruStory/octopus/services/truapi/db"
)

func (ta *TruAPI) AppAccountLoader() *AppAccountLoader {
	ctx := ta.createContext(context.Background())
	config := AppAccountLoaderConfig{
		Fetch: func(keys []string) ([]*AppAccount, []error) {
			errors := make([]error, 0)
			addresses := make([]sdk.AccAddress, 0, len(keys))
			for _, k := range keys {
				address, err := sdk.AccAddressFromBech32(k)
				if err != nil {
					errors = append(errors, err)
					return nil, errors
				}
				addresses = append(addresses, address)
			}
			accounts, err := ta.appAccountsResolver(ctx, addresses)
			if err != nil {
				errors = append(errors, err)
				return nil, errors
			}
			return accounts, nil
		},
		Wait:     time.Millisecond * 2,
		MaxBatch: 50,
	}
	return NewAppAccountLoader(config)
}

func (ta *TruAPI) UserProfileLoader() *UserProfileLoader {
	config := UserProfileLoaderConfig{
		Fetch: func(addresses []string) ([]*db.UserProfile, []error) {
			users, err := ta.DBClient.UsersByAddress(addresses)
			errors := make([]error, 0)
			if err != nil {
				errors = append(errors, err)
				return nil, errors
			}
			mappedUsers := make(map[string]db.User, len(addresses))
			for _, user := range users {
				mappedUsers[user.Address] = user
			}
			output := make([]*db.UserProfile, len(addresses))
			for i, address := range addresses {
				user, ok := mappedUsers[address]
				if !ok {
					continue
				}
				output[i] = &db.UserProfile{
					FullName:  user.FullName,
					Bio:       user.Bio,
					AvatarURL: user.AvatarURL,
					Username:  user.Username,
				}
			}
			return output, nil
		},
		Wait:     time.Millisecond * 2,
		MaxBatch: 50,
	}
	return NewUserProfileLoader(config)
}
