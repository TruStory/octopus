package shifters

import "fmt"

type MixpanelShifter struct{}

func (s MixpanelShifter) Shift(r Replacers) error {
	fmt.Println("shifting mixpanel")

	return nil
}
