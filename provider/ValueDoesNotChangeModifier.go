package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

type ValueDoesNotChangeModifier struct{}

func (m ValueDoesNotChangeModifier) Description(ctx context.Context) string {
	return "Value does not change after creation."
}

func (m ValueDoesNotChangeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m ValueDoesNotChangeModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
	if req.State.Raw.IsNull() {
		// if we're creating the resource, no need to delete and
		// recreate it
		return
	}

	if req.Plan.Raw.IsNull() {
		// if we're deleting the resource, no need to delete and
		// recreate it
		return
	}

	resp.Diagnostics.AddAttributeError(req.AttributePath, "Value cannot be changed", "This attribute is blocked for updating")
	return

}
