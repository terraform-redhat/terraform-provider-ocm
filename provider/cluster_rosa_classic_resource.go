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
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"net/http"
	"net/url"
	"strings"
)

const (
	awsCloudProvider = "aws"
	rosaProduct      = "rosa"
)

type ClusterRosaClassicResourceType struct {
}

type ClusterRosaClassicResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *ClusterRosaClassicResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "OpenShift managed cluster.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "Unique identifier of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
			"external_id": {
				Description: "Unique external identifier of the cluster.",
				Type:        types.StringType,
				Computed:    true,
			},
			"name": {
				Description: "Name of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"cloud_region": {
				Description: "Cloud region identifier, for example 'us-east-1'.",
				Type:        types.StringType,
				Required:    true,
			},
			"sts": {
				Description: "STS Configuration",
				Attributes:  StsResource(),
				Optional:    true,
			},
			"multi_az": {
				Description: "Indicates if the cluster should be deployed to " +
					"multiple availability zones. Default value is 'false'.",
				Type:     types.BoolType,
				Optional: true,
				Computed: true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.RequiresReplace(),
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
			"compute_nodes": {
				Description: "Number of compute nodes of the cluster.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
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
			"aws_account_id": {
				Description: "Identifier of the AWS account.",
				Type:        types.StringType,
				Required:    true,
			},
			"aws_subnet_ids": {
				Description: "aws subnet ids",
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"aws_private_link": {
				Description: "aws subnet ids",
				Type:        types.BoolType,
				Optional:    true,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					tfsdk.RequiresReplace(),
				},
			},
			"availability_zones": {
				Description: "availability zones",
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"machine_cidr": {
				Description: "Block of IP addresses for nodes.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
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
				}),
				Optional: true,
			},
			"service_cidr": {
				Description: "Block of IP addresses for services.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"pod_cidr": {
				Description: "Block of IP addresses for pods.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"host_prefix": {
				Description: "Length of the prefix of the subnet assigned to each node.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
			},
			"version": {
				Description: "Identifier of the version of OpenShift, for example 'openshift-v4.1.0'.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
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

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &ClusterRosaClassicResource{
		logger:     parent.logger,
		collection: collection,
	}

	return
}

func (r *ClusterRosaClassicResource) Create(ctx context.Context,
	request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	r.logger.Info(ctx, "********************  Create 1   **********************")
	// Get the plan:
	state := &ClusterRosaClassicState{}
	diags := request.Plan.Get(ctx, state)
	for _, diag := range diags {
		r.logger.Info(ctx, "********************  Diag detail   %s   **********************", diag.Detail())
		r.logger.Info(ctx, "********************  Diag summary   %s   **********************", diag.Summary())
		r.logger.Info(ctx, "********************  Diag summary   %s   **********************", diag.Summary())
	}
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	r.logger.Info(ctx, "********************  Create 2   **********************")
	// Create the cluster:
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

	nodes := cmv1.NewClusterNodes()
	if !state.ComputeNodes.Unknown && !state.ComputeNodes.Null {
		nodes.Compute(int(state.ComputeNodes.Value))
	}
	if !state.ComputeMachineType.Unknown && !state.ComputeMachineType.Null {
		nodes.ComputeMachineType(
			cmv1.NewMachineType().ID(state.ComputeMachineType.Value),
		)
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
	builder.CCS(ccs)

	aws := cmv1.NewAWS()
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

	sts := cmv1.NewSTS()
	if state.Sts != nil {
		sts.RoleARN(state.Sts.RoleARN.Value)
		sts.SupportRoleARN(state.Sts.SupportRoleArn.Value)
		instanceIamRoles := cmv1.NewInstanceIAMRoles()
		instanceIamRoles.MasterRoleARN(state.Sts.InstanceIAMRoles.MasterRoleARN.Value)
		instanceIamRoles.WorkerRoleARN(state.Sts.InstanceIAMRoles.WorkerRoleARN.Value)
		sts.InstanceIAMRoles(instanceIamRoles)

		r.logger.Info(ctx, "******************** Create 3 OperatorRolePrefix  **********************")
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
		builder.Version(cmv1.NewVersion().ID(state.Version.Value))
	}

	proxy := cmv1.NewProxy()
	if state.Proxy != nil {
		proxy.HTTPProxy(state.Proxy.HttpProxy.Value)
		proxy.HTTPSProxy(state.Proxy.HttpsProxy.Value)
		builder.Proxy(proxy)
	}
	object, err := builder.Build()
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
	r.populateState(ctx, object, state)
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
	r.populateState(ctx, object, state)
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
	builder := cmv1.NewCluster()
	var nodes *cmv1.ClusterNodesBuilder
	compute, ok := ShouldPatchInt(state.ComputeNodes, plan.ComputeNodes)
	if ok {
		nodes.Compute(int(compute))
	}
	if !nodes.Empty() {
		builder.Nodes(nodes)
	}
	patch, err := builder.Build()
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
		Body(patch).
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
	object := update.Body()

	// Update the state:
	r.populateState(ctx, object, state)
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
	r.populateState(ctx, object, state)
	diags := response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

// populateState copies the data from the API object to the Terraform state.
func (r *ClusterRosaClassicResource) populateState(ctx context.Context, object *cmv1.Cluster, state *ClusterRosaClassicState) {
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
	state.ComputeNodes = types.Int64{
		Value: int64(object.Nodes().Compute()),
	}
	state.ComputeMachineType = types.String{
		Value: object.Nodes().ComputeMachineType().ID(),
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

	state.EtcdEncryption = types.Bool{
		Value: object.EtcdEncryption(),
	}

	autoscaleCompute := object.Nodes().AutoscaleCompute()
	if autoscaleCompute != nil {
		state.AutoScalingEnabled = types.Bool{
			Value: true,
		}

		state.MaxReplicas = types.Int64{
			Value: int64(autoscaleCompute.MaxReplicas()),
		}

		state.MinReplicas = types.Int64{
			Value: int64(autoscaleCompute.MinReplicas()),
		}
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

	sts, ok := object.AWS().GetSTS()
	if ok {
		state.Sts = &Sts{}
		oidc_endpoint_url := sts.OIDCEndpointURL()
		r.logger.Info(ctx, "******************** populate  OIDCEndpointURL %s **********************", oidc_endpoint_url)
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
		state.Sts.InstanceIAMRoles.MasterRoleARN = types.String{
			Value: sts.InstanceIAMRoles().MasterRoleARN(),
		}
		state.Sts.InstanceIAMRoles.WorkerRoleARN = types.String{
			Value: sts.InstanceIAMRoles().WorkerRoleARN(),
		}

		thumbprint, err := getThumbprint(sts.OIDCEndpointURL())
		if err != nil {
			r.logger.Error(ctx, "cannot get thumbprint", err)
			state.Sts.Thumbprint = types.String{
				Value: "",
			}
		} else {
			state.Sts.Thumbprint = types.String{
				Value: thumbprint,
			}
		}

		// TODO:  add validation!!!
		computedOperatorRoles := []OperatorIAMRole{}
		for _, operatorRole := range sts.OperatorIAMRoles() {
			r.logger.Info(ctx, "******************** populate  operator role %s, %s, %s **********************", operatorRole.Name(), operatorRole.Namespace(), operatorRole.Namespace())
			tempRole := OperatorIAMRole{
				Name: types.String{
					Value: operatorRole.Name(),
				},
				Namespace: types.String{
					Value: operatorRole.Namespace(),
				},
				RoleARN: types.String{
					Value: operatorRole.RoleARN(),
				},
			}
			computedOperatorRoles = append(computedOperatorRoles, tempRole)
		}
		state.Sts.OperatorIAMRoles = &computedOperatorRoles
		r.logger.Info(ctx, "******************** populate  OperatorRolePrefix %s **********************", sts.OperatorRolePrefix())
		state.Sts.OperatorRolePrefix = types.String{
			Value: sts.OperatorRolePrefix(),
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

	r.logger.Info(ctx, "******************** populate 5 OperatorRolePrefix value: %s **********************", state.Sts.OperatorRolePrefix.Value)
}

func getThumbprint(oidcEndpointURL string) (string, error) {
	connect, err := url.ParseRequestURI(oidcEndpointURL)
	if err != nil {
		return "", err
	}

	response, err := http.Get(fmt.Sprintf("https://%s:443", connect.Host))
	if err != nil {
		return "", err
	}

	certChain := response.TLS.PeerCertificates

	// Grab the CA in the chain
	for _, cert := range certChain {
		if cert.IsCA {
			if bytes.Equal(cert.RawIssuer, cert.RawSubject) {
				return sha1Hash(cert.Raw), nil
			}
		}
	}

	// Fall back to using the last certficiate in the chain
	cert := certChain[len(certChain)-1]
	return sha1Hash(cert.Raw), nil
}

// sha1Hash computes the SHA1 of the byte array and returns the hex encoding as a string.
func sha1Hash(data []byte) string {
	// nolint:gosec
	hasher := sha1.New()
	hasher.Write(data)
	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed)
}
