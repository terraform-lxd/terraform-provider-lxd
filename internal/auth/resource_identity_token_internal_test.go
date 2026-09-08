package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpiryPattern(t *testing.T) {
	tests := []struct {
		expiry  string
		isValid bool
	}{
		{
			expiry:  "30d",
			isValid: true,
		},
		{
			expiry:  "1H 30M",
			isValid: true,
		},
		{
			expiry:  "0H 5M",
			isValid: true,
		},
		{
			expiry:  "0S",
			isValid: false,
		},
		{
			expiry:  "0H 0M",
			isValid: false,
		},
		{
			expiry:  "00S",
			isValid: false,
		},
		{
			expiry:  "",
			isValid: false,
		},
		{
			expiry:  "1H30M",
			isValid: false,
		},
		{
			expiry:  "30x",
			isValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.expiry, func(t *testing.T) {
			t.Parallel()
			// The schema applies both validators, so an expiry is accepted only
			// when it is a valid duration list and is greater than zero.
			valid := expiryPattern.MatchString(test.expiry) && expiryNonZeroPattern.MatchString(test.expiry)
			require.Equal(t, test.isValid, valid)
		})
	}
}
