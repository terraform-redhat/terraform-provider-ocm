/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm_cluster_rosa_classic

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ClusterRosaClassicState struct {
	APIURL             types.String `tfsdk:"api_url"`
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`
	AWSAccountID       types.String `tfsdk:"aws_account_id"`
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"`
	AWSSubnetIDs       types.List   `tfsdk:"aws_subnet_ids"`
	AWSPrivateLink     types.Bool   `tfsdk:"aws_private_link"`
	Sts                *Sts         `tfsdk:"sts"`
	CCSEnabled         types.Bool   `tfsdk:"ccs_enabled"`
	CloudRegion        types.String `tfsdk:"cloud_region"`
	ComputeMachineType types.String `tfsdk:"compute_machine_type"`
	ComputeNodes       types.Int64  `tfsdk:"compute_nodes"`
	ConsoleURL         types.String `tfsdk:"console_url"`
	HostPrefix         types.Int64  `tfsdk:"host_prefix"`
	ID                 types.String `tfsdk:"id"`
	MachineCIDR        types.String `tfsdk:"machine_cidr"`
	MultiAZ            types.Bool   `tfsdk:"multi_az"`
	AvailabilityZones  types.List   `tfsdk:"availability_zones"`
	Name               types.String `tfsdk:"name"`
	PodCIDR            types.String `tfsdk:"pod_cidr"`
	Properties         types.Map    `tfsdk:"properties"`
	ServiceCIDR        types.String `tfsdk:"service_cidr"`
	Proxy              *Proxy       `tfsdk:"proxy"`
	State              types.String `tfsdk:"state"`
	Version            types.String `tfsdk:"version"`
	Wait               types.Bool   `tfsdk:"wait"`
}

type Sts struct {
	OIDCEndpointURL  types.String      `tfsdk:"oidc_endpoint_url"`
	Thumbprint       types.String      `tfsdk:"thumbprint"`
	RoleARN          types.String      `tfsdk:"role_arn"`
	SupportRoleArn   types.String      `tfsdk:"support_role_arn"`
	InstanceIAMRoles InstanceIAMRole   `tfsdk:"instance_iam_roles"`
	OperatorIAMRoles []OperatorIAMRole `tfsdk:"operator_iam_roles"`
}

type InstanceIAMRole struct {
	MasterRoleARN types.String `tfsdk:"master_role_arn"`
	WorkerRoleARN types.String `tfsdk:"worker_role_arn"`
}

type OperatorIAMRole struct {
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
	RoleARN   types.String `tfsdk:"role_arn"`
}

type Proxy struct {
	HttpProxy  types.String `tfsdk:"http_proxy"`
	HttpsProxy types.String `tfsdk:"https_proxy"`
	NoProxy    types.String `tfsdk:"no_proxy"`
}
