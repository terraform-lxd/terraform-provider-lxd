package cluster

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestIsPendingClusterLink(t *testing.T) {
	tests := []struct {
		name      string
		addresses types.List
		expected  bool
	}{
		{"no addresses recorded", types.ListValueMust(types.StringType, []attr.Value{}), true},
		{"addresses recorded", types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.1:8443")}), false},
		{"addresses not yet read", types.ListNull(types.StringType), false},
		{"addresses not yet known", types.ListUnknown(types.StringType), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.expected, isPendingClusterLink(test.addresses))
		})
	}
}
