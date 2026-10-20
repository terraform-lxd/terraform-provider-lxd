package common

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestTokenExpiry(t *testing.T) {
	// LXD returns a zero time when core.remote_token_expiry is disabled.
	require.True(t, TokenExpiry(time.Time{}).IsNull())

	expiresAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-07-30T12:00:00Z", TokenExpiry(expiresAt).ValueString())
}

func TestIsTokenExpired(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt types.String
		expected  bool
	}{
		{"expired token", types.StringValue(now.Add(-time.Minute).Format(time.RFC3339)), true},
		{"valid token", types.StringValue(now.Add(time.Minute).Format(time.RFC3339)), false},
		{"token that never expires", types.StringNull(), false},
		{"expiry not yet known", types.StringUnknown(), false},
		{"unparsable expiry", types.StringValue("2026/07/30 12:00 UTC"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, IsTokenExpired(test.expiresAt, now))
		})
	}
}
