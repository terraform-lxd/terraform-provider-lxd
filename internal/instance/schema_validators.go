package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/maveonair/terraform-provider-incus/internal/utils"
)

// configKeyValidator ensures config key does not start
// with "volatile.", "image.", or "limits.".
type configKeyValidator struct{}

func (v configKeyValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("config key cannot have %q, %q, or %q prefix", "volatile.", "image.", "limits.")
}
func (v configKeyValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("config key cannot have `%s`, `%s`, or `%s` prefix", "volatile.", "image.", "limits.")
}

func (v configKeyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	value := req.ConfigValue.ValueString()

	if strings.HasPrefix(value, "limits.") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config key",
			fmt.Sprintf("Config keys with %q prefix must be provided in %q map instead.", "limits.", "limits"),
		)
	}

	if utils.HasAnyPrefix(value, []string{"volatile.", "image."}) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config key",
			fmt.Sprintf("Config key cannot have %q or %q prefix. Got: %q.", "volatile.", "image.", value),
		)
	}
}
