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

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ClusterRosaClassicState struct {
	APIURL             types.String `tfsdk:"api_url"`
	AWSAccountID       types.String `tfsdk:"aws_account_id"`
	AWSSubnetIDs       types.List   `tfsdk:"aws_subnet_ids"`
	AWSPrivateLink     types.Bool   `tfsdk:"aws_private_link"`
	Sts                *Sts         `tfsdk:"sts"`
	CCSEnabled         types.Bool   `tfsdk:"ccs_enabled"`
	EtcdEncryption     types.Bool   `tfsdk:"etcd_encryption"`
	AutoScalingEnabled types.Bool   `tfsdk:"autoscaling_enabled"`
	MinReplicas        types.Int64  `tfsdk:"min_replicas"`
	MaxReplicas        types.Int64  `tfsdk:"max_replicas"`
	CloudRegion        types.String `tfsdk:"cloud_region"`
	ComputeMachineType types.String `tfsdk:"compute_machine_type"`
	ComputeNodes       types.Int64  `tfsdk:"compute_nodes"`
	ConsoleURL         types.String `tfsdk:"console_url"`
	HostPrefix         types.Int64  `tfsdk:"host_prefix"`
	ID                 types.String `tfsdk:"id"`
	ExternalID         types.String `tfsdk:"external_id"`
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
	//OperatorIAMRoles   types.List   `tfsdk:"operator_iam_roles"`
	OperatorIAMRoles OperatorIAMRoleInfo `tfsdk:"operator_iam_roles"`
}

type Sts struct {
	OIDCEndpointURL    types.String    `tfsdk:"oidc_endpoint_url"`
	Thumbprint         types.String    `tfsdk:"thumbprint"`
	RoleARN            types.String    `tfsdk:"role_arn"`
	SupportRoleArn     types.String    `tfsdk:"support_role_arn"`
	InstanceIAMRoles   InstanceIAMRole `tfsdk:"instance_iam_roles"`
	OperatorRolePrefix types.String    `tfsdk:"operator_role_prefix"`
	//OperatorIAMRoles   *OperatorIAMRoles `tfsdk:"operator_iam_roles"`
	//OperatorIAMRoles   []OperatorIAMRoleInfo `tfsdk:"operator_iam_roles"`
	//OperatorIAMRoles []types.Object `tfsdk:"operator_iam_roles"`
}

type InstanceIAMRole struct {
	MasterRoleARN types.String `tfsdk:"master_role_arn"`
	WorkerRoleARN types.String `tfsdk:"worker_role_arn"`
}

type OperatorIAMRoles struct {
	CloudCredentialRole *OperatorIAMRoleInfo `tfsdk:"cloud_credential_role"`
	CloudNetworkConfig  *OperatorIAMRoleInfo `tfsdk:"cloud_network_config"`
	CsiDriversRole      *OperatorIAMRoleInfo `tfsdk:"csi_drivers_role"`
	ImageRegistryRole   *OperatorIAMRoleInfo `tfsdk:"image_registry_role"`
	IngressOperatorRole *OperatorIAMRoleInfo `tfsdk:"ingress_operator_role"`
	MachineApiRole      *OperatorIAMRoleInfo `tfsdk:"machine_api_role"`
}

type OperatorIAMRoleInfo struct {
	Name            types.String `tfsdk:"name"`
	Namespace       types.String `tfsdk:"namespace"`
	OperatorRoleARN types.String `tfsdk:"operator_role_arn"`
}
