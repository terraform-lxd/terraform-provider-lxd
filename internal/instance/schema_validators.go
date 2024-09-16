package instance

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/lxc/terraform-provider-incus/internal/utils"
)

// configKeyValidator ensures config key does not start
// with "volatile." or "image.".
type configKeyValidator struct{}

func (v configKeyValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("config key cannot have %q or %q prefix", "volatile.", "image.")
}
func (v configKeyValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("config key cannot have `%s` or `%s` prefix", "volatile.", "image.")
}

func (v configKeyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	value := req.ConfigValue.ValueString()

	if utils.HasAnyPrefix(value, []string{"volatile.", "image."}) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config key",
			fmt.Sprintf("Config key cannot have %q or %q prefix. Got: %q.", "volatile.", "image.", value),
		)
	}
}
