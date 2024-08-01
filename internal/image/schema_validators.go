package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared/osarch"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type architectureValidator struct{}

func (v architectureValidator) Description(ctx context.Context) string {
	supportedArchitecturesList := strings.Join(osarch.SupportedArchitectures(), ", ")
	return fmt.Sprintf("Attribute architecture value must be one of: %s.", supportedArchitecturesList)
}

func (v architectureValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v architectureValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	for _, supportedArchitecture := range osarch.SupportedArchitectures() {
		if value == supportedArchitecture {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(req.Path, "Invalid architecture",
		v.Description(ctx),
	)
}
