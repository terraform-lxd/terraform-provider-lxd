package storage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// configSourceValidator ensures storage pool source in configuration is treated as
// read-only attribute.
type configSourceValidator struct{}

func (v configSourceValidator) Description(_ context.Context) string {
	return fmt.Sprintf("config key %q is read-only", "source")
}
func (v configSourceValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("config key `%s` is read-only", "source")
}

func (v configSourceValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	value := req.ConfigValue.ValueString()

	if value == "source" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid config key", fmt.Sprint(
			`Setting storage pool source using "config.source" is not allowed, `+
				`as it will produce an inconsistent Terraform plan. Use "source" `+
				`attribute instead which will be respected only during the `+
				`creation of the storage pool.`),
		)
	}
}
