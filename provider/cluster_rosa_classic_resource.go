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
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	semver "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocm_errors "github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

const (
	awsCloudProvider = "aws"
	rosaProduct      = "rosa"
	MinVersion       = "4.10"
)

var kmsArnRE = regexp.MustCompile(
	`^arn:aws[\w-]*:kms:[\w-]+:\d{12}:key\/mrk-[0-9a-f]{32}$|[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

type ClusterRosaClassicResourceType struct {
	logger logging.Logger
}

type ClusterRosaClassicResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *ClusterRosaClassicResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "OpenShift managed cluster using rosa sts.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "Unique identifier of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
			"external_id": {
				Description: "Unique external identifier of the cluster.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"name": {
				Description: "Name of the cluster.",
				Type:        types.StringType,
				Required:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"cloud_region": {
				Description: "Cloud region identifier, for example 'us-east-1'.",
				Type:        types.StringType,
				Required:    true,
			},
			"sts": {
				Description: "STS Configuration",
				Attributes:  stsResource(),
				Optional:    true,
			},
			"multi_az": {
				Description: "Indicates if the cluster should be deployed to " +
					"multiple availability zones. Default value is 'false'.",
				Type:     types.BoolType,
				Optional: true,
				Computed: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"disable_workload_monitoring": {
				Description: "Enables you to monitor your own projects in isolation from Red Hat " +
					"Site Reliability Engineer (SRE) platform metrics.",
				Type:     types.BoolType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"disable_scp_checks": {
				Description: "Enables you to monitor your own projects in isolation from Red Hat " +
					"Site Reliability Engineer (SRE) platform metrics.",
				Type:     types.BoolType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"properties": {
				Description: "User defined properties.",
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
				Computed: true,
			},
			"tags": {
				Description: "Apply user defined tags to all resources created in AWS.",
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"ccs_enabled": {
				Description: "Enables customer cloud subscription.",
				Type:        types.BoolType,
				Computed:    true,
			},
			"etcd_encryption": {
				Description: "Encrypt etcd data.",
				Type:        types.BoolType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"autoscaling_enabled": {
				Description: "Enables autoscaling.",
				Type:        types.BoolType,
				Optional:    true,
			},
			"min_replicas": {
				Description: "Min replicas.",
				Type:        types.Int64Type,
				Optional:    true,
			},
			"max_replicas": {
				Description: "Max replicas.",
				Type:        types.Int64Type,
				Optional:    true,
			},
			"api_url": {
				Description: "URL of the API server.",
				Type:        types.StringType,
				Computed:    true,
			},
			"console_url": {
				Description: "URL of the console.",
				Type:        types.StringType,
				Computed:    true,
			},
			"replicas": {
				Description: "Number of worker nodes to provision. Single zone clusters need at least 2 nodes, " +
					"multizone clusters need at least 3 nodes.",
				Type:     types.Int64Type,
				Optional: true,
				Computed: true,
			},
			"compute_machine_type": {
				Description: "Identifier of the machine type used by the compute nodes, " +
					"for example `r5.xlarge`. Use the `ocm_machine_types` data " +
					"source to find the possible values.",
				Type:     types.StringType,
				Optional: true,
				Computed: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.RequiresReplace(),
				},
			},
			"compute_labels": {
				Description: "Labels for the default machine pool. Format should be a comma-separated list of 'key=value'. " +
					"This list will overwrite any modifications made to Node labels on an ongoing basis.",
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"aws_account_id": {
				Description: "Identifier of the AWS account.",
				Type:        types.StringType,
				Required:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"aws_subnet_ids": {
				Description: "aws subnet ids",
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"kms_key_arn": {
				Description: "The key ARN is the Amazon Resource Name (ARN) of a AWS KMS (Key Management Service) Key. It is a unique, " +
					"fully qualified identifier for the AWS KMS Key. A key ARN includes the AWS account, Region, and the key ID.",
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"fips": {
				Description: "Create cluster that uses FIPS Validated / Modules in Process cryptographic libraries",
				Type:        types.BoolType,
				Optional:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"aws_private_link": {
				Description: "aws subnet ids",
				Type:        types.BoolType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"availability_zones": {
				Description: "availability zones",
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"machine_cidr": {
				Description: "Block of IP addresses for nodes.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"proxy": {
				Description: "proxy",
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"http_proxy": {
						Description: "http proxy",
						Type:        types.StringType,
						Required:    true,
					},
					"https_proxy": {
						Description: "https proxy",
						Type:        types.StringType,
						Required:    true,
					},
					"no_proxy": {
						Description: "no proxy",
						Type:        types.StringType,
						Optional:    true,
					},
					"additional_trust_bundle": {
						Description: "a string contains contains a PEM-encoded X.509 certificate bundle that will be added to the nodes' trusted certificate store.",
						Type:        types.StringType,
						Optional:    true,
					},
				}),
				Optional: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"service_cidr": {
				Description: "Block of IP addresses for services.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"pod_cidr": {
				Description: "Block of IP addresses for pods.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"host_prefix": {
				Description: "Length of the prefix of the subnet assigned to each node.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"version": {
				Description: "Identifier of the version of OpenShift, for example 'openshift-v4.1.0'.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				// TODO: till AWS will support Managed policies we will not support update versions
				PlanModifiers: []tfsdk.AttributePlanModifier{
					ValueCannotBeChangedModifier(t.logger),
				},
			},
			"disable_waiting_in_destroy": {
				Description: "Disable addressing cluster state in the destroy resource. Default value is false",
				Type:        types.BoolType,
				Optional:    true,
			},
			"destroy_timeout": {
				Description: "Timeout in minutes for addressing cluster state in destroy resource. Default value is 60 minutes.",
				Type:        types.Int64Type,
				Optional:    true,
			},
			"state": {
				Description: "State of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
		},
	}
	return
}

func (t *ClusterRosaClassicResourceType) NewResource(ctx context.Context,
	p tfsdk.Provider) (result tfsdk.Resource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation:
	parent := p.(*Provider)

	// Get the collection:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &ClusterRosaClassicResource{
		logger:     parent.logger,
		collection: collection,
	}

	return
}

const (
	errHeadline = "Can't build cluster"
)

func createClassicClusterObject(ctx context.Context,
	state *ClusterRosaClassicState, logger logging.Logger, diags diag.Diagnostics) (*cmv1.Cluster, error) {

	builder := cmv1.NewCluster()
	builder.Name(state.Name.Value)
	builder.CloudProvider(cmv1.NewCloudProvider().ID(awsCloudProvider))
	builder.Product(cmv1.NewProduct().ID(rosaProduct))
	builder.Region(cmv1.NewCloudRegion().ID(state.CloudRegion.Value))
	if !state.MultiAZ.Unknown && !state.MultiAZ.Null {
		builder.MultiAZ(state.MultiAZ.Value)
	}
	if !state.Properties.Unknown && !state.Properties.Null {
		properties := map[string]string{}
		for k, v := range state.Properties.Elems {
			properties[k] = v.(types.String).Value
		}
		builder.Properties(properties)
	}

	if !state.EtcdEncryption.Unknown && !state.EtcdEncryption.Null {
		builder.EtcdEncryption(state.EtcdEncryption.Value)
	}

	if !state.ExternalID.Unknown && !state.ExternalID.Null {
		builder.ExternalID(state.ExternalID.Value)
	}

	if !state.DisableWorkloadMonitoring.Unknown && !state.DisableWorkloadMonitoring.Null {
		builder.DisableUserWorkloadMonitoring(state.DisableWorkloadMonitoring.Value)
	}

	nodes := cmv1.NewClusterNodes()
	if !state.Replicas.Unknown && !state.Replicas.Null {
		nodes.Compute(int(state.Replicas.Value))
	}
	if !state.ComputeMachineType.Unknown && !state.ComputeMachineType.Null {
		nodes.ComputeMachineType(
			cmv1.NewMachineType().ID(state.ComputeMachineType.Value),
		)
	}

	if !state.ComputeLabels.Unknown && !state.ComputeLabels.Null {
		labels := map[string]string{}
		for k, v := range state.ComputeLabels.Elems {
			labels[k] = v.(types.String).Value
		}
		nodes.ComputeLabels(labels)
	}

	if !state.AvailabilityZones.Unknown && !state.AvailabilityZones.Null {
		azs := make([]string, 0)
		for _, e := range state.AvailabilityZones.Elems {
			azs = append(azs, e.(types.String).Value)
		}
		nodes.AvailabilityZones(azs...)
	}

	if !state.AutoScalingEnabled.Unknown && !state.AutoScalingEnabled.Null && state.AutoScalingEnabled.Value {
		autoscaling := cmv1.NewMachinePoolAutoscaling()
		if !state.MaxReplicas.Unknown && !state.MaxReplicas.Null {
			autoscaling.MaxReplicas(int(state.MaxReplicas.Value))
		}
		if !state.MinReplicas.Unknown && !state.MinReplicas.Null {
			autoscaling.MinReplicas(int(state.MinReplicas.Value))
		}
		if !autoscaling.Empty() {
			nodes.AutoscaleCompute(autoscaling)
		}
	}

	if !nodes.Empty() {
		builder.Nodes(nodes)
	}

	// ccs should be enabled in ocm rosa clusters
	ccs := cmv1.NewCCS()
	ccs.Enabled(true)

	if !state.DisableSCPChecks.Unknown && !state.DisableSCPChecks.Null && state.DisableSCPChecks.Value {
		ccs.DisableSCPChecks(true)
	}
	builder.CCS(ccs)

	aws := cmv1.NewAWS()

	if !state.Tags.Unknown && !state.Tags.Null {
		tags := map[string]string{}
		for k, v := range state.Tags.Elems {
			if _, ok := tags[k]; ok {
				errDescription := fmt.Sprintf("Invalid tags, user tag keys must be unique, duplicate key '%s' found", k)
				logger.Error(ctx, errDescription)

				diags.AddError(
					errHeadline,
					errDescription,
				)
				return nil, errors.New(errHeadline + "\n" + errDescription)
			}
			tags[k] = v.(types.String).Value
		}

		aws.Tags(tags)
	}

	if !state.KMSKeyArn.Unknown && !state.KMSKeyArn.Null && state.KMSKeyArn.Value != "" {
		kmsKeyARN := state.KMSKeyArn.Value
		if !kmsArnRE.MatchString(kmsKeyARN) {
			errDescription := fmt.Sprintf("Expected a valid value for kms-key-arn matching %s", kmsArnRE)
			logger.Error(ctx, errDescription)

			diags.AddError(
				errHeadline,
				errDescription,
			)
			return nil, errors.New(errHeadline + "\n" + errDescription)
		}
		aws.KMSKeyArn(kmsKeyARN)
	}

	if !state.AWSAccountID.Unknown && !state.AWSAccountID.Null {
		aws.AccountID(state.AWSAccountID.Value)
	}

	if !state.AWSPrivateLink.Unknown && !state.AWSPrivateLink.Null {
		aws.PrivateLink((state.AWSPrivateLink.Value))
		api := cmv1.NewClusterAPI()
		if state.AWSPrivateLink.Value {
			api.Listening(cmv1.ListeningMethodInternal)
		}
		builder.API(api)
	}

	if !state.FIPS.Unknown && !state.FIPS.Null && state.FIPS.Value {
		builder.FIPS(true)
	}

	sts := cmv1.NewSTS()
	if state.Sts != nil {
		sts.RoleARN(state.Sts.RoleARN.Value)
		sts.SupportRoleARN(state.Sts.SupportRoleArn.Value)
		instanceIamRoles := cmv1.NewInstanceIAMRoles()
		instanceIamRoles.MasterRoleARN(state.Sts.InstanceIAMRoles.MasterRoleARN.Value)
		instanceIamRoles.WorkerRoleARN(state.Sts.InstanceIAMRoles.WorkerRoleARN.Value)
		sts.InstanceIAMRoles(instanceIamRoles)

		sts, err := checkAndSetByoOidcAttributes(ctx, state, sts)
		if err != nil {
			logger.Error(ctx, fmt.Sprintf("%v", err))
			return nil, err
		}

		sts.OperatorRolePrefix(state.Sts.OperatorRolePrefix.Value)
		aws.STS(sts)
	}

	if !state.AWSSubnetIDs.Unknown && !state.AWSSubnetIDs.Null {
		subnetIds := make([]string, 0)
		for _, e := range state.AWSSubnetIDs.Elems {
			subnetIds = append(subnetIds, e.(types.String).Value)
		}
		aws.SubnetIDs(subnetIds...)
	}

	if !aws.Empty() {
		builder.AWS(aws)
	}
	network := cmv1.NewNetwork()
	if !state.MachineCIDR.Unknown && !state.MachineCIDR.Null {
		network.MachineCIDR(state.MachineCIDR.Value)
	}
	if !state.ServiceCIDR.Unknown && !state.ServiceCIDR.Null {
		network.ServiceCIDR(state.ServiceCIDR.Value)
	}
	if !state.PodCIDR.Unknown && !state.PodCIDR.Null {
		network.PodCIDR(state.PodCIDR.Value)
	}
	if !state.HostPrefix.Unknown && !state.HostPrefix.Null {
		network.HostPrefix(int(state.HostPrefix.Value))
	}
	if !network.Empty() {
		builder.Network(network)
	}
	if !state.Version.Unknown && !state.Version.Null {
		// TODO: update it to support all cluster versions
		isSupported, err := checkSupportedVersion(state.Version.Value)
		if err != nil {
			logger.Error(ctx, "Error validating required cluster version %s\", err)")
			errDecription := fmt.Sprintf(
				"Can't check if cluster version is supported '%s': %v",
				state.Version.Value, err,
			)
			diags.AddError(
				errHeadline,
				errDecription,
			)
			return nil, errors.New(errHeadline + "\n" + errDecription)
		}
		if isSupported {
			builder.Version(cmv1.NewVersion().ID(state.Version.Value))
		} else {
			logger.Error(ctx, "Cluster version %s is not supported", state.Version.Value)
			errDecription := fmt.Sprintf(
				"Can't check if cluster version is supported '%s': %v",
				state.Version.Value, err,
			)
			diags.AddError(
				errHeadline,
				errDecription,
			)
			return nil, errors.New(errHeadline + "\n" + errDecription)
		}
	}

	proxy := cmv1.NewProxy()
	if state.Proxy != nil {
		proxy.HTTPProxy(state.Proxy.HttpProxy.Value)
		proxy.HTTPSProxy(state.Proxy.HttpsProxy.Value)
		if !state.Proxy.AdditionalTrustBundle.Unknown && !state.Proxy.AdditionalTrustBundle.Null {
			builder.AdditionalTrustBundle(state.Proxy.AdditionalTrustBundle.Value)
		}
		builder.Proxy(proxy)
	}

	object, err := builder.Build()
	return object, err
}

func checkAndSetByoOidcAttributes(ctx context.Context, state *ClusterRosaClassicState, sts *cmv1.STSBuilder) (*cmv1.STSBuilder, error) {
	isByoOidcSet := isByoOidcSet(state.Sts)
	if isByoOidcSet {
		if state.Sts.OIDCEndpointURL.Unknown || state.Sts.OIDCEndpointURL.Null || state.Sts.OIDCEndpointURL.Value == "" {
			errDescription := fmt.Sprintf("When using BYO OIDC Endpoint URL cannot be empty")
			return nil, errors.New(errHeadline + "\n" + errDescription)
		}
		if state.Sts.OIDCPrivateKeySecretArn.Unknown || state.Sts.OIDCPrivateKeySecretArn.Null || state.Sts.OIDCPrivateKeySecretArn.Value == "" {
			errDescription := fmt.Sprintf("When using BYO OIDC Secret ARN cannot be empty")
			return nil, errors.New(errHeadline + "\n" + errDescription)
		}
		sts.OIDCEndpointURL("https://" + state.Sts.OIDCEndpointURL.Value)
		sts.OidcPrivateKeySecretArn(state.Sts.OIDCPrivateKeySecretArn.Value)
	}
	return sts, nil
}

func isByoOidcSet(sts *Sts) bool {
	return !sts.OIDCEndpointURL.Unknown && !sts.OIDCEndpointURL.Null && sts.OIDCEndpointURL.Value != "" ||
		!sts.OIDCPrivateKeySecretArn.Unknown && !sts.OIDCPrivateKeySecretArn.Null && sts.OIDCPrivateKeySecretArn.Value != ""
}

func (r *ClusterRosaClassicResource) Create(ctx context.Context,
	request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	// Get the plan:
	state := &ClusterRosaClassicState{}
	diags := request.Plan.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	object, err := createClassicClusterObject(ctx, state, r.logger, diags)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build cluster",
			fmt.Sprintf(
				"Can't build cluster with name '%s': %v",
				state.Name.Value, err,
			),
		)
		return
	}
	add, err := r.collection.Add().Body(object).SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't create cluster",
			fmt.Sprintf(
				"Can't create cluster with name '%s': %v",
				state.Name.Value, err,
			),
		)
		return
	}
	object = add.Body()

	// Save the state:
	err = populateRosaClassicClusterState(ctx, object, state, r.logger, DefaultHttpClient{})
	if err != nil {
		response.Diagnostics.AddError(
			"Can't populate cluster state",
			fmt.Sprintf(
				"Received error %v", err,
			),
		)
		return
	}
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterRosaClassicResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest,
	response *tfsdk.ReadResourceResponse) {
	// Get the current state:
	state := &ClusterRosaClassicState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Find the cluster:
	get, err := r.collection.Cluster(state.ID.Value).Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find cluster",
			fmt.Sprintf(
				"Can't find cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	object := get.Body()

	// Save the state:
	err = populateRosaClassicClusterState(ctx, object, state, r.logger, DefaultHttpClient{})
	if err != nil {
		response.Diagnostics.AddError(
			"Can't populate cluster state",
			fmt.Sprintf(
				"Received error %v", err,
			),
		)
		return
	}
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterRosaClassicResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest,
	response *tfsdk.UpdateResourceResponse) {
	var diags diag.Diagnostics

	// Get the state:
	state := &ClusterRosaClassicState{}
	diags = request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Get the plan:
	plan := &ClusterRosaClassicState{}
	diags = request.Plan.Get(ctx, plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send request to update the cluster:
	updateNodes := false
	clusterBuilder := cmv1.NewCluster()
	clusterNodesBuilder := cmv1.NewClusterNodes()
	compute, ok := shouldPatchInt(state.Replicas, plan.Replicas)
	if ok {
		clusterNodesBuilder = clusterNodesBuilder.Compute(int(compute))
		updateNodes = true
	}

	if !plan.AutoScalingEnabled.Unknown && !plan.AutoScalingEnabled.Null && plan.AutoScalingEnabled.Value {
		// autoscaling enabled
		autoscaling := cmv1.NewMachinePoolAutoscaling()

		if !plan.MaxReplicas.Unknown && !plan.MaxReplicas.Null {
			autoscaling = autoscaling.MaxReplicas(int(plan.MaxReplicas.Value))
		}
		if !plan.MinReplicas.Unknown && !plan.MinReplicas.Null {
			autoscaling = autoscaling.MinReplicas(int(plan.MinReplicas.Value))
		}

		clusterNodesBuilder = clusterNodesBuilder.AutoscaleCompute(autoscaling)
		updateNodes = true

	} else {
		if (!plan.MaxReplicas.Unknown && !plan.MaxReplicas.Null) || (!plan.MinReplicas.Unknown && !plan.MinReplicas.Null) {
			response.Diagnostics.AddError(
				"Can't update cluster",
				fmt.Sprintf(
					"Can't update MaxReplica and/or MinReplica of cluster when autoscaling is not enabled",
				),
			)
			return
		}
	}

	if updateNodes {
		clusterBuilder = clusterBuilder.Nodes(clusterNodesBuilder)
	}
	clusterSpec, err := clusterBuilder.Build()
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build cluster patch",
			fmt.Sprintf(
				"Can't build patch for cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	update, err := r.collection.Cluster(state.ID.Value).Update().
		Body(clusterSpec).
		SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't update cluster",
			fmt.Sprintf(
				"Can't update cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}

	// update the autoscaling enabled with the plan value (important for nil and false cases)
	state.AutoScalingEnabled = plan.AutoScalingEnabled
	// update the Replicas with the plan value (important for nil and zero value cases)
	state.Replicas = plan.Replicas

	object := update.Body()

	// Update the state:
	err = populateRosaClassicClusterState(ctx, object, state, r.logger, DefaultHttpClient{})
	if err != nil {
		response.Diagnostics.AddError(
			"Can't populate cluster state",
			fmt.Sprintf(
				"Received error %v", err,
			),
		)
		return
	}
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterRosaClassicResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest,
	response *tfsdk.DeleteResourceResponse) {
	// Get the state:
	state := &ClusterRosaClassicState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send the request to delete the cluster:
	resource := r.collection.Cluster(state.ID.Value)
	_, err := resource.Delete().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't delete cluster",
			fmt.Sprintf(
				"Can't delete cluster with identifier '%s': %v",
				state.ID.Value, err,
			),
		)
		return
	}
	if !state.DisableWaitingInDestroy.Unknown && !state.DisableWaitingInDestroy.Null && state.DisableWaitingInDestroy.Value {
		r.logger.Info(ctx, "Waiting for destroy to be completed, is disabled")
	} else {
		timeout := defaultTimeoutInMinutes
		if !state.DestroyTimeout.Unknown && !state.DestroyTimeout.Null {
			if state.DestroyTimeout.Value <= 0 {
				response.Diagnostics.AddWarning(nonPositiveTimeoutSummary, fmt.Sprintf(nonPositiveTimeoutFormat, state.ID.Value))
			} else {
				timeout = state.DestroyTimeout.Value
			}
		}
		isNotFound, err := r.retryClusterNotFoundWithTimeout(3, 1*time.Minute, ctx, timeout, resource)
		if err != nil {
			response.Diagnostics.AddError(
				"Can't poll cluster state",
				fmt.Sprintf(
					"Can't poll state of cluster with identifier '%s': %v",
					state.ID.Value, err,
				),
			)
			return
		}

		if !isNotFound {
			response.Diagnostics.AddWarning(
				"Cluster wasn't deleted yet",
				fmt.Sprintf("The cluster with identifier '%s' is not deleted yet, but the polling finisehd due to a timeout", state.ID.Value),
			)
		}

	}
	// Remove the state:
	response.State.RemoveResource(ctx)
}

func (r *ClusterRosaClassicResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest,
	response *tfsdk.ImportResourceStateResponse) {
	// Try to retrieve the object:
	get, err := r.collection.Cluster(request.ID).Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find cluster",
			fmt.Sprintf(
				"Can't find cluster with identifier '%s': %v",
				request.ID, err,
			),
		)
		return
	}
	object := get.Body()

	// Save the state:
	state := &ClusterRosaClassicState{}
	err = populateRosaClassicClusterState(ctx, object, state, r.logger, DefaultHttpClient{})
	if err != nil {
		response.Diagnostics.AddError(
			"Can't populate cluster state",
			fmt.Sprintf(
				"Received error %v", err,
			),
		)
		return
	}

	diags := response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

// populateRosaClassicClusterState copies the data from the API object to the Terraform state.
func populateRosaClassicClusterState(ctx context.Context, object *cmv1.Cluster, state *ClusterRosaClassicState, logger logging.Logger, httpClient HttpClient) error {
	state.ID = types.String{
		Value: object.ID(),
	}
	state.ExternalID = types.String{
		Value: object.ExternalID(),
	}
	object.API()
	state.Name = types.String{
		Value: object.Name(),
	}
	state.CloudRegion = types.String{
		Value: object.Region().ID(),
	}
	state.MultiAZ = types.Bool{
		Value: object.MultiAZ(),
	}
	state.Properties = types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for k, v := range object.Properties() {
		state.Properties.Elems[k] = types.String{
			Value: v,
		}
	}
	state.APIURL = types.String{
		Value: object.API().URL(),
	}
	state.ConsoleURL = types.String{
		Value: object.Console().URL(),
	}
	state.Replicas = types.Int64{
		Value: int64(object.Nodes().Compute()),
	}
	state.ComputeMachineType = types.String{
		Value: object.Nodes().ComputeMachineType().ID(),
	}

	labels, ok := object.Nodes().GetComputeLabels()
	if ok {
		state.ComputeLabels = types.Map{
			ElemType: types.StringType,
			Elems:    map[string]attr.Value{},
		}
		for k, v := range labels {
			state.ComputeLabels.Elems[k] = types.String{
				Value: v,
			}
		}
	}

	disableUserWorkload, ok := object.GetDisableUserWorkloadMonitoring()
	if ok && disableUserWorkload {
		state.DisableWorkloadMonitoring = types.Bool{
			Value: true,
		}
	}

	isFips, ok := object.GetFIPS()
	if ok && isFips {
		state.FIPS = types.Bool{
			Value: true,
		}
	}
	autoScaleCompute, ok := object.Nodes().GetAutoscaleCompute()
	if ok {
		var maxReplicas, minReplicas int
		state.AutoScalingEnabled = types.Bool{
			Value: true,
		}

		maxReplicas, ok = autoScaleCompute.GetMaxReplicas()
		if ok {
			state.MaxReplicas = types.Int64{
				Value: int64(maxReplicas),
			}
		}

		minReplicas, ok = autoScaleCompute.GetMinReplicas()
		if ok {
			state.MinReplicas = types.Int64{
				Value: int64(minReplicas),
			}
		}
	} else {
		// autoscaling not enabled - initialize the MaxReplica and MinReplica
		state.MaxReplicas.Null = true
		state.MinReplicas.Null = true
	}

	azs, ok := object.Nodes().GetAvailabilityZones()
	if ok {
		state.AvailabilityZones.Elems = make([]attr.Value, 0)
		for _, az := range azs {
			state.AvailabilityZones.Elems = append(state.AvailabilityZones.Elems, types.String{
				Value: az,
			})
		}
	}

	state.CCSEnabled = types.Bool{
		Value: object.CCS().Enabled(),
	}

	disableSCPChecks, ok := object.CCS().GetDisableSCPChecks()
	if ok && disableSCPChecks {
		state.DisableSCPChecks = types.Bool{
			Value: true,
		}
	}

	state.EtcdEncryption = types.Bool{
		Value: object.EtcdEncryption(),
	}

	//The API does not return account id
	awsAccountID, ok := object.AWS().GetAccountID()
	if ok {
		state.AWSAccountID = types.String{
			Value: awsAccountID,
		}
	}

	awsPrivateLink, ok := object.AWS().GetPrivateLink()
	if ok {
		state.AWSPrivateLink = types.Bool{
			Value: awsPrivateLink,
		}
	} else {
		state.AWSPrivateLink = types.Bool{
			Null: true,
		}
	}
	kmsKeyArn, ok := object.AWS().GetKMSKeyArn()
	if ok {
		state.KMSKeyArn = types.String{
			Value: kmsKeyArn,
		}
	}

	sts, ok := object.AWS().GetSTS()
	if ok {
		if state.Sts == nil {
			state.Sts = &Sts{}
		}
		oidc_endpoint_url := sts.OIDCEndpointURL()
		if strings.HasPrefix(oidc_endpoint_url, "https://") {
			oidc_endpoint_url = strings.TrimPrefix(oidc_endpoint_url, "https://")
		}

		state.Sts.OIDCEndpointURL = types.String{
			Value: oidc_endpoint_url,
		}
		state.Sts.RoleARN = types.String{
			Value: sts.RoleARN(),
		}
		state.Sts.SupportRoleArn = types.String{
			Value: sts.SupportRoleARN(),
		}
		instanceIAMRoles := sts.InstanceIAMRoles()
		if instanceIAMRoles != nil {
			state.Sts.InstanceIAMRoles.MasterRoleARN = types.String{
				Value: instanceIAMRoles.MasterRoleARN(),
			}
			state.Sts.InstanceIAMRoles.WorkerRoleARN = types.String{
				Value: instanceIAMRoles.WorkerRoleARN(),
			}

		}
		// TODO: fix a bug in uhc-cluster-services
		if state.Sts.OperatorRolePrefix.Unknown || state.Sts.OperatorRolePrefix.Null {
			operatorRolePrefix, ok := sts.GetOperatorRolePrefix()
			if ok {
				state.Sts.OperatorRolePrefix = types.String{
					Value: operatorRolePrefix,
				}
			}
		}
		thumbprint, err := getThumbprint(sts.OIDCEndpointURL(), httpClient)
		if err != nil {
			logger.Error(ctx, "cannot get thumbprint", err)
			state.Sts.Thumbprint = types.String{
				Value: "",
			}
		} else {
			state.Sts.Thumbprint = types.String{
				Value: thumbprint,
			}
		}
	}

	subnetIds, ok := object.AWS().GetSubnetIDs()
	if ok {
		state.AWSSubnetIDs.Elems = make([]attr.Value, 0)
		for _, subnetId := range subnetIds {
			state.AWSSubnetIDs.Elems = append(state.AWSSubnetIDs.Elems, types.String{
				Value: subnetId,
			})
		}
	}

	proxy, ok := object.GetProxy()
	if ok {
		state.Proxy.HttpProxy = types.String{
			Value: proxy.HTTPProxy(),
		}
		state.Proxy.HttpsProxy = types.String{
			Value: proxy.HTTPSProxy(),
		}
	}

	trustBundle, ok := object.GetAdditionalTrustBundle()
	if ok {
		state.Proxy.AdditionalTrustBundle = types.String{
			Value: trustBundle,
		}
	}

	machineCIDR, ok := object.Network().GetMachineCIDR()
	if ok {
		state.MachineCIDR = types.String{
			Value: machineCIDR,
		}
	} else {
		state.MachineCIDR = types.String{
			Null: true,
		}
	}
	serviceCIDR, ok := object.Network().GetServiceCIDR()
	if ok {
		state.ServiceCIDR = types.String{
			Value: serviceCIDR,
		}
	} else {
		state.ServiceCIDR = types.String{
			Null: true,
		}
	}
	podCIDR, ok := object.Network().GetPodCIDR()
	if ok {
		state.PodCIDR = types.String{
			Value: podCIDR,
		}
	} else {
		state.PodCIDR = types.String{
			Null: true,
		}
	}
	hostPrefix, ok := object.Network().GetHostPrefix()
	if ok {
		state.HostPrefix = types.Int64{
			Value: int64(hostPrefix),
		}
	} else {
		state.HostPrefix = types.Int64{
			Null: true,
		}
	}
	version, ok := object.Version().GetID()
	if ok {
		state.Version = types.String{
			Value: version,
		}
	} else {
		state.Version = types.String{
			Null: true,
		}
	}
	state.State = types.String{
		Value: string(object.State()),
	}

	return nil
}

type HttpClient interface {
	Get(url string) (resp *http.Response, err error)
}

type DefaultHttpClient struct {
}

func (c DefaultHttpClient) Get(url string) (resp *http.Response, err error) {
	return http.Get(url)
}

func getThumbprint(oidcEndpointURL string, httpClient HttpClient) (thumbprint string, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			fmt.Fprintf(os.Stderr, "recovering from: %q\n", panicErr)
			thumbprint = ""
			err = fmt.Errorf("recovering from: %q", panicErr)
		}
	}()

	connect, err := url.ParseRequestURI(oidcEndpointURL)
	if err != nil {
		return "", err
	}

	response, err := httpClient.Get(fmt.Sprintf("https://%s:443", connect.Host))
	if err != nil {
		return "", err
	}

	certChain := response.TLS.PeerCertificates

	// Grab the CA in the chain
	for _, cert := range certChain {
		if cert.IsCA {
			if bytes.Equal(cert.RawIssuer, cert.RawSubject) {
				hash, err := sha1Hash(cert.Raw)
				if err != nil {
					return "", err
				}
				return hash, nil
			}
		}
	}

	// Fall back to using the last certficiate in the chain
	cert := certChain[len(certChain)-1]
	return sha1Hash(cert.Raw)
}

// sha1Hash computes the SHA1 of the byte array and returns the hex encoding as a string.
func sha1Hash(data []byte) (string, error) {
	// nolint:gosec
	hasher := sha1.New()
	_, err := hasher.Write(data)
	if err != nil {
		return "", fmt.Errorf("Couldn't calculate hash:\n %v", err)
	}
	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed), nil
}

func checkSupportedVersion(clusterVersion string) (bool, error) {
	rawID := strings.Replace(clusterVersion, "openshift-v", "", 1)
	v1, err := semver.NewVersion(rawID)
	if err != nil {
		return false, err
	}
	v2, err := semver.NewVersion(MinVersion)
	if err != nil {
		return false, err
	}
	//Cluster version is greater than or equal to MinVersion
	return v1.GreaterThanOrEqual(v2), nil
}

func (r *ClusterRosaClassicResource) retryClusterNotFoundWithTimeout(attempts int, sleep time.Duration, ctx context.Context, timeout int64,
	resource *cmv1.ClusterClient) (bool, error) {
	isNotFound, err := r.waitTillClusterIsNotFoundWithTimeout(ctx, timeout, resource, r.logger)
	if err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return r.retryClusterNotFoundWithTimeout(attempts, 2*sleep, ctx, timeout, resource)
		}
		return isNotFound, err
	}

	return isNotFound, nil
}

func (r *ClusterRosaClassicResource) waitTillClusterIsNotFoundWithTimeout(ctx context.Context, timeout int64,
	resource *cmv1.ClusterClient, logger logging.Logger) (bool, error) {
	timeoutInMinutes := time.Duration(timeout) * time.Minute
	pollCtx, cancel := context.WithTimeout(ctx, timeoutInMinutes)
	defer cancel()
	_, err := resource.Poll().
		Interval(pollingIntervalInMinutes * time.Minute).
		Status(http.StatusNotFound).
		StartContext(pollCtx)
	sdkErr, ok := err.(*ocm_errors.Error)
	if ok && sdkErr.Status() == http.StatusNotFound {
		logger.Info(ctx, "Cluster was removed")
		return true, nil
	}
	if err != nil {
		logger.Error(ctx, "Can't poll cluster deletion")
		return false, err
	}

	return false, nil
}
