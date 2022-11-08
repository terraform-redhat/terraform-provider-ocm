package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func StsResource() tfsdk.NestedAttributes {
	return tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
		"oidc_endpoint_url": {
			Description: "OIDC Endpoint URL",
			Type:        types.StringType,
			Computed:    true,
		},
		"thumbprint": {
			Description: "SHA1-hash value of the root CA of the issuer URL",
			Type:        types.StringType,
			Computed:    true,
		},
		"role_arn": {
			Description: "Installer Role",
			Type:        types.StringType,
			Required:    true,
		},
		"support_role_arn": {
			Description: "Support Role",
			Type:        types.StringType,
			Required:    true,
		},
		"instance_iam_roles": {
			Description: "Instance IAm Roles",
			Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
				"master_role_arn": {
					Description: "Master/Controller Plane Role ARN",
					Type:        types.StringType,
					Required:    true,
				},
				"worker_role_arn": {
					Description: "Worker Node Role ARN",
					Type:        types.StringType,
					Required:    true,
				},
			}),
			Required: true,
		},
		"operator_role_prefix": {
			Description: "prefix for operator role ",
			Type:        types.StringType,
			Required:    true,
		},
		"operator_iam_roles": {
			Description: "Operator IAM Roles",
			Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
				"name": {
					Description: "Operator Name",
					Type:        types.StringType,
					Computed:    true,
				},
				"namespace": {
					Description: "Kubernetes Namespace",
					Type:        types.StringType,
					Computed:    true,
				},
				"role_arn": {
					Description: "AWS Role ARN",
					Type:        types.StringType,
					Computed:    true,
				},
			}, tfsdk.ListNestedAttributesOptions{
				MinItems: 6,
				MaxItems: 6}),
			Computed: true,
		},
	})
}
