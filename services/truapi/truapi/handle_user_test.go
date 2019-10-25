package truapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEmailDomain(t *testing.T) {
	testCases := []struct {
		desc           string
		email          string
		expectedDomain string
	}{
		{
			desc:           "Empty email",
			email:          "",
			expectedDomain: "",
		},
		{
			desc:           "Regular email",
			email:          "test.super@gmail.com",
			expectedDomain: "gmail.com",
		},
		{
			desc:           "Test with spaces",
			email:          "  test.super@gmail.com     ",
			expectedDomain: "gmail.com",
		},
		{
			desc:           "Test multiple @",
			email:          "  test.super@mydomain@thisistherealdomain.net",
			expectedDomain: "thisistherealdomain.net",
		},
		{
			desc:           "Test multiple @@",
			email:          "  test@.+test@super@mydomain@thisistherealdomain.net",
			expectedDomain: "thisistherealdomain.net",
		},
		{
			desc:           "Test whithout @",
			email:          "  test.random+test.com",
			expectedDomain: "",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expectedDomain, getEmailDomain(tC.email))
		})
	}
}
