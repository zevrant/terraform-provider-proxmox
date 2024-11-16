package proxmox

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

type sshKeyListValidator struct{}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (validator sshKeyListValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("ssh keys must only contain the key and not and labels")
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (validator sshKeyListValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("ssh keys must only contain the key and not and labels")
}

func (validator sshKeyListValidator) ValidateList(ctx context.Context, request validator.ListRequest, response *validator.ListResponse) {
	if request.ConfigValue.IsUnknown() || request.ConfigValue.IsNull() {
		return //don't care if not provided
	}

	keysList := make([]types.String, 0, len(request.ConfigValue.Elements()))
	_ = request.ConfigValue.ElementsAs(ctx, &keysList, false)

	for _, key := range keysList {
		if len(strings.Split(key.ValueString(), " ")) != 2 {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Malformed SSH Key Entered",
				"Multiple areas of whitespace detected, a properly formed key should only contain one space between the algorithm declaration and the actual key",
			)
			return
		}
	}

}
