package auth

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestTokenExpiry(t *testing.T) {
	// LXD returns a zero time when core.remote_token_expiry is disabled.
	require.True(t, tokenExpiry(time.Time{}).IsNull())

	expiresAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-07-30T12:00:00Z", tokenExpiry(expiresAt).ValueString())
}

func TestIsTrustTokenExpired(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	past := types.StringValue(now.Add(-time.Minute).Format(time.RFC3339))
	future := types.StringValue(now.Add(time.Minute).Format(time.RFC3339))

	tests := []struct {
		name      string
		isPending bool
		expiresAt types.String
		expected  bool
	}{
		{
			name:      "pending identity with an expired token",
			isPending: true,
			expiresAt: past,
			expected:  true,
		},
		{
			name:      "pending identity with a valid token",
			isPending: true,
			expiresAt: future,
			expected:  false,
		},
		{
			name:      "pending identity with a token that never expires",
			isPending: true,
			expiresAt: types.StringNull(),
			expected:  false,
		},
		{
			name:      "pending identity with an unparsable expiry",
			isPending: true,
			expiresAt: types.StringValue("2026/07/30 12:00 UTC"),
			expected:  false,
		},
		{
			name:      "redeemed identity keeps a stale expiry",
			isPending: false,
			expiresAt: past,
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, isTrustTokenExpired(test.isPending, test.expiresAt, now))
		})
	}
}

func TestIsCertificateSetOnPending(t *testing.T) {
	certificate := types.StringValue("-----BEGIN CERTIFICATE-----")

	tests := []struct {
		name              string
		isPending         bool
		configCertificate types.String
		expected          bool
	}{
		{
			name:              "pending identity with a configured certificate",
			isPending:         true,
			configCertificate: certificate,
			expected:          true,
		},
		{
			name:              "pending identity with a certificate that is not yet computed",
			isPending:         true,
			configCertificate: types.StringUnknown(),
			expected:          true,
		},
		{
			name:              "pending identity without a configured certificate",
			isPending:         true,
			configCertificate: types.StringNull(),
			expected:          false,
		},
		{
			name:              "redeemed identity rotating its certificate",
			isPending:         false,
			configCertificate: certificate,
			expected:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, isCertificateSetOnPending(test.isPending, test.configCertificate))
		})
	}
}
