package shifters

import (
	"fmt"
	"strings"

	"github.com/dukex/mixpanel"
)

type MixpanelShifter struct {
	Token string
}

func (s MixpanelShifter) Shift(r Replacers) error {
	client := mixpanel.New(s.Token, "")

	for _, replacer := range r {
		// omitting the replacers that are not cosmos addresses
		if !strings.HasPrefix(replacer.From, "cosmos") {
			continue
		}

		// creating the alias
		err := client.Alias(replacer.From, replacer.To)
		if err != nil {
			return err
		}

		// tracking the change on the new alias (to test as well as to document the change)
		err = client.Track(replacer.To, "Alias Shifted", &mixpanel.Event{
			Properties: map[string]interface{}{
				"Previous Address": replacer.From,
			},
		})
		if err != nil {
			return err
		}
		fmt.Print(".")
	}
	fmt.Print("DONE.")
	return nil
}
