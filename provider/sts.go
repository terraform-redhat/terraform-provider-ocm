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

		//"operator_iam_roles": {
		//	Description: "Operator IAM Roles",
		//	Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
		//		"name": {
		//			Type:     types.StringType,
		//			Computed: true,
		//			PlanModifiers: tfsdk.AttributePlanModifiers{
		//				tfsdk.RequiresReplace(),
		//			},
		//		},
		//	}, tfsdk.ListNestedAttributesOptions{}),
		//	PlanModifiers: tfsdk.AttributePlanModifiers{
		//		tfsdk.RequiresReplace(),
		//	},
		//	Computed: true,
		//Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
		//	"cloud_credential_role": {
		//		Description: "Cloud Credential Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//	"cloud_network_config": {
		//		Description: "Cloud Network Config Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//	"csi_drivers_role": {
		//		Description: "Csi Drivers Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//	"image_registry_role": {
		//		Description: "Image Registry Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//	"ingress_operator_role": {
		//		Description: "Ingress Operator Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//	"machine_api_role": {
		//		Description: "Machine Api Role",
		//		Attributes:  OperatorRoleInfo(),
		//		Optional:    true,
		//	},
		//}),
		//PlanModifiers: tfsdk.AttributePlanModifiers{
		//	tfsdk.UseStateForUnknown(),
		//},
		//},
	})
}

func OperatorRoleInfo() map[string]tfsdk.Attribute {
	return map[string]tfsdk.Attribute{
		"name": {
			Description: "operator role name",
			Type:        types.StringType,
			Computed:    true,
		},
		"namespace": {
			Description: "operator role namespace",
			Type:        types.StringType,
			Computed:    true,
		},
		"operator_role_arn": {
			Description: "operator role arn",
			Type:        types.StringType,
			Computed:    true,
		},
	}
}
