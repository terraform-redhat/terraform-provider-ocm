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

package clusterresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/balancer/grpclb/state"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/terraform-provider-ocm/provider/util"
)

func ResourceCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		UpdateContext: resourceClusterUpdate,
		DeleteContext: resourceClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "Unique identifier of the cluster.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"product": {
				Description: "Product ID OSD or Rosa",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "Name of the cluster.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"cloud_provider": {
				Description: "Cloud provider identifier, for example 'aws'.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"cloud_region": {
				Description: "Cloud region identifier, for example 'us-east-1'.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"multi_az": {
				Description: "Indicates if the cluster should be deployed to " +
					"multiple availability zones. Default value is 'false'.",
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"properties": {
				Description: "User defined properties.",
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Computed:    true,
			},
			"api_url": {
				Description: "URL of the API server.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"console_url": {
				Description: "URL of the console.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"compute_nodes": {
				Description: "Number of compute nodes of the cluster.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"compute_machine_type": {
				Description: "Identifier of the machine type used by the compute nodes, " +
					"for example `r5.xlarge`. Use the `ocm_machine_types` data " +
					"source to find the possible values.",
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"ccs_enabled": {
				Description: "Enables customer cloud subscription.",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"aws_account_id": {
				Description: "Identifier of the AWS account.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"aws_access_key_id": {
				Description: "Identifier of the AWS access key.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"aws_secret_access_key": {
				Description: "AWS access key.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"aws_subnet_ids": {
				Description: "aws subnet ids",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"aws_private_link": {
				Description: "aws subnet ids",
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
			},
			"availability_zones": {
				Description: "availability zones",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"machine_cidr": {
				Description: "Block of IP addresses for nodes.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"proxy": {
				Description: "proxy",
				Type:        schema.TypeList,
				MaxItems: 1,
				Computed:    true,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"http_proxy": {
							Description: "http proxy",
							Type:        schema.TypeString,
							Required:    true,
						},
						"https_proxy": {
							Description: "https proxy",
							Type:        schema.TypeString,
							Required:    true,
						},
						"no_proxy": {
							Description: "no proxy",
							Type:        schema.TypeString,
							Optional:    true,
						},
					}},
			},
			"service_cidr": {
				Description: "Block of IP addresses for services.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"pod_cidr": {
				Description: "Block of IP addresses for pods.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"host_prefix": {
				Description: "Length of the prefix of the subnet assigned to each node.",
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
			},
			"version": {
				Description: "Identifier of the version of OpenShift, for example 'openshift-v4.1.0'.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"state": {
				Description: "State of the cluster.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"wait": {
				Description: "Wait till the cluster is ready.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func resourceClusterCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connection := meta.(*sdk.Connection)
	state := getClusterStateFromResourceData(resourceData)
	cluster, err := createClusterObject(ctx, state)
	if err != nil {
		return diag.Errorf("Can't build cluster", err.Error())
	}

	add, err := connection.ClustersMgmt().V1().Clusters().Add().Body(cluster).SendContext(ctx)
	if err != nil {
		return diag.Errorf("Can't create cluster", err.Error())
	}
	cluster = add.Body()

	// Wait till the cluster is ready unless explicitly disabled:
	if state.Wait != nil {
		ready := cluster.State() == cmv1.ClusterStateReady
		if *state.Wait && !ready {
			pollCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
			defer cancel()
			_, err := connection.ClustersMgmt().V1().Clusters().Cluster(cluster.ID()).Poll().
				Interval(30 * time.Second).
				Predicate(func(get *cmv1.ClusterGetResponse) bool {
					cluster = get.Body()
					return cluster.State() == cmv1.ClusterStateReady
				}).
				StartContext(pollCtx)
			if err != nil {
				return diag.Errorf("Can't poll cluster state", err.Error())
			}
		}
	}

	return resourceClusterRead(ctx, resourceData, meta)
}

func getClusterStateFromResourceData(resourceData *schema.ResourceData) *ClusterState {
	state := &ClusterState{
		Product:       resourceData.Get("product").(string),
		Name:          resourceData.Get("name").(string),
		CloudProvider: resourceData.Get("cloud_provider").(string),
		CloudRegion:   resourceData.Get("cloud_region").(string),
		APIURL:        resourceData.Get("api_url").(string),
		ConsoleURL:    resourceData.Get("console_url").(string),
	}

	if v, ok := resourceData.GetOk("multi_az"); ok {
		multiAz := v.(bool)
		state.MultiAZ = &multiAz
	}

	if v, ok := resourceData.GetOk("properties"); ok {
		properties := util.ExpandStringMap(v.(map[string]interface{}))
		state.Properties = &properties
	}

	if v, ok := resourceData.GetOk("compute_nodes"); ok {
		computeNodes := int64(v.(int))
		state.ComputeNodes = &computeNodes
	}

	if v, ok := resourceData.GetOk("compute_machine_type"); ok {
		computeMachineType := v.(string)
		state.ComputeMachineType = &computeMachineType
	}

	if v, ok := resourceData.GetOk("ccs_enabled"); ok {
		ccsEnabled := v.(bool)
		state.CCSEnabled = &ccsEnabled
	}

	if v, ok := resourceData.GetOk("aws_account_id"); ok {
		awsAccountID := v.(string)
		state.AWSAccountID = &awsAccountID
	}

	if v, ok := resourceData.GetOk("aws_access_key_id"); ok {
		awsAccessKeyID := v.(string)
		state.AWSAccessKeyID = &awsAccessKeyID
	}

	if v, ok := resourceData.GetOk("aws_secret_access_key"); ok {
		awsSecretAccessKey := v.(string)
		state.AWSSecretAccessKey = &awsSecretAccessKey
	}

	if v, ok := resourceData.GetOk("aws_subnet_ids"); ok {
		awsSubnetIDs := util.ExpandStringValueList(v.([]interface{}))
		state.AWSSubnetIDs = &awsSubnetIDs
	}

	if v, ok := resourceData.GetOk("aws_private_link"); ok {
		awsPrivateLink := v.(bool)
		state.AWSPrivateLink = &awsPrivateLink
	}

	if v, ok := resourceData.GetOk("availability_zones"); ok {
		availabilityZones := util.ExpandStringValueList(v.([]interface{}))
		state.AvailabilityZones = &availabilityZones
	}

	if v, ok := resourceData.GetOk("machine_cidr"); ok {
		machineCIDR := v.(string)
		state.MachineCIDR = &machineCIDR
	}

	if v, ok := resourceData.GetOk("service_cidr"); ok {
		serviceCIDR := v.(string)
		state.ServiceCIDR = &serviceCIDR
	}

	if v, ok := resourceData.GetOk("pod_cidr"); ok {
		podCIDR := v.(string)
		state.PodCIDR = &podCIDR
	}

	if v, ok := resourceData.GetOk("host_prefix"); ok {
		hostPrefix := int64(v.(int))
		state.HostPrefix = &hostPrefix
	}

	if v, ok := resourceData.GetOk("version"); ok {
		version := v.(string)
		state.Version = &version
	}

	if v, ok := resourceData.GetOk("wait"); ok {
		wait := v.(bool)
		state.Wait = &wait
	}

	if v, ok := resourceData.GetOk("proxy"); ok && len(v.([]interface{})) > 0 {
		proxy := v.([]interface{})[0]
		proxyMap := util.ExpandStringMap(proxy.(map[string]interface{}))
		state.Proxy = &Proxy{
			HttpProxy:  proxyMap["http_proxy"],
			HttpsProxy: proxyMap["https_proxy"],
		}

		if noProxy, ok := proxyMap["no_proxy"]; ok {
			state.Proxy.NoProxy = noProxy
		}
	}

	return state
}

func resourceClusterRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connection := meta.(*sdk.Connection)
	util.Logger.Info(ctx, "[INFO] Reading cluster")
	clusterID := resourceData.Id()
	object, err := connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Get().SendContext(ctx)
	cluster := object.Body()
	if err != nil {
		return diag.Errorf("Can't find cluster", err.Error())
	}

	return populateClusterState(cluster, resourceData)
}
func resourceClusterUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connection := meta.(*sdk.Connection)
	builder := cmv1.NewCluster()
	var nodes *cmv1.ClusterNodesBuilder

	if resourceData.HasChange("compute_nodes") {
		computeNodes := resourceData.Get("compute_nodes").(int)
		nodes.Compute(computeNodes)
	}

	if !nodes.Empty() {
		builder.Nodes(nodes)
	}

	patch, err := builder.Build()
	if err != nil {
		return diag.Errorf("Can't build cluster patch", err.Error())
	}
	update, err := connection.ClustersMgmt().V1().Clusters().Cluster(resourceData.Id()).Update().
		Body(patch).
		SendContext(ctx)
	if err != nil {
		return diag.Errorf("Can't update cluster", err.Error())
	}

	cluster := update.Body()
	util.Logger.Info(ctx, "[INFO] Updated cluster %s", cluster.ID())

	return resourceClusterRead(ctx, resourceData, meta)
}
func resourceClusterDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connection := meta.(*sdk.Connection)
	resource := connection.ClustersMgmt().V1().Clusters().Cluster(resourceData.Id())
	_, err := resource.Delete().SendContext(ctx)

	if err != nil {
		return diag.Errorf("Can't delete cluster", err.Error())
	}
	state := getClusterStateFromResourceData(resourceData)

	// Wait till the cluster has been effectively deleted:
	if state.Wait != nil {
		pollCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		_, err = resource.Poll().
			Interval(30 * time.Second).
			Status(http.StatusNotFound).
			StartContext(pollCtx)
		sdkErr, ok := err.(*errors.Error)
		if ok && sdkErr.Status() == http.StatusNotFound {
			err = nil
		}
		if err != nil {
			return diag.Errorf("Can't poll cluster deletion", err.Error())
		}
	}

	err = resourceClusterRead(ctx, resourceData, meta)
	if err != nil {
		return diag.Errorf("Can't set the state for resourceData", err.Error())
	}
	return nil
}

func createClusterObject(_ context.Context,
	state *ClusterState) (*cmv1.Cluster, error) {
	// Create the cluster:
	builder := cmv1.NewCluster()
	builder.Name(state.Name)
	builder.CloudProvider(cmv1.NewCloudProvider().ID(state.CloudProvider))
	builder.Product(cmv1.NewProduct().ID(state.Product))
	builder.Region(cmv1.NewCloudRegion().ID(state.CloudRegion))
	if state.MultiAZ != nil {
		builder.MultiAZ(*state.MultiAZ)
	}
	if state.Properties != nil && len(*state.Properties) > 0 {
		builder.Properties(*state.Properties)
	}

	nodes := cmv1.NewClusterNodes()
	if state.ComputeNodes != nil {
		nodes.Compute(int(*state.ComputeNodes))
	}

	if state.ComputeMachineType != nil {
		nodes.ComputeMachineType(
			cmv1.NewMachineType().ID(*state.ComputeMachineType),
		)
	}

	if state.AvailabilityZones != nil {
		azs := make([]string, 0)
		for _, e := range *state.AvailabilityZones {
			azs = append(azs, e)
		}
		nodes.AvailabilityZones(azs...)
	}

	if !nodes.Empty() {
		builder.Nodes(nodes)
	}
	ccs := cmv1.NewCCS()
	if state.CCSEnabled != nil {
		ccs.Enabled(*state.CCSEnabled)
	}
	if !ccs.Empty() {
		builder.CCS(ccs)
	}
	aws := cmv1.NewAWS()
	if state.AWSAccountID != nil {
		aws.AccountID(*state.AWSAccountID)
	}
	if state.AWSAccessKeyID != nil {
		aws.AccessKeyID(*state.AWSAccessKeyID)
	}
	if state.AWSSecretAccessKey != nil {
		aws.SecretAccessKey(*state.AWSSecretAccessKey)
	}
	if state.AWSPrivateLink != nil {
		aws.PrivateLink((*state.AWSPrivateLink))
		api := cmv1.NewClusterAPI()
		if *state.AWSPrivateLink {
			api.Listening(cmv1.ListeningMethodInternal)
		}
		builder.API(api)
	}

	if state.AWSSubnetIDs != nil {
		subnetIds := make([]string, 0)
		for _, e := range *state.AWSSubnetIDs {
			subnetIds = append(subnetIds, e)
		}
		aws.SubnetIDs(subnetIds...)
	}

	if !aws.Empty() {
		builder.AWS(aws)
	}
	network := cmv1.NewNetwork()
	if state.MachineCIDR != nil {
		network.MachineCIDR(*state.MachineCIDR)
	}
	if state.ServiceCIDR != nil {
		network.ServiceCIDR(*state.ServiceCIDR)
	}
	if state.PodCIDR != nil {
		network.PodCIDR(*state.PodCIDR)
	}
	if state.HostPrefix != nil {
		network.HostPrefix(int(*state.HostPrefix))
	}
	if !network.Empty() {
		builder.Network(network)
	}
	if state.Version != nil {
		builder.Version(cmv1.NewVersion().ID(*state.Version))
	}

	proxy := cmv1.NewProxy()
	if state.Proxy != nil {
		proxy.HTTPProxy(state.Proxy.HttpProxy)
		proxy.HTTPSProxy(state.Proxy.HttpsProxy)
		builder.Proxy(proxy)
	}

	object, err := builder.Build()

	return object, err
}

// populateClusterState copies the data from the API object to the Terraform state.
func populateClusterState(cluster *cmv1.Cluster, resourceData *schema.ResourceData) error {
	if err := resourceData.Set("id", cluster.ID()); err != nil {
		return err
	}

	if err := resourceData.Set("product", cluster.Product().ID()); err != nil {
		return err
	}

	if err := resourceData.Set("name", cluster.Name()); err != nil {
		return err
	}

	if err := resourceData.Set("cloud_provider", cluster.CloudProvider().ID()); err != nil {
		return err
	}

	if err := resourceData.Set("cloud_region", cluster.Region().ID()); err != nil {
		return err
	}

	if err := resourceData.Set("multi_az", cluster.MultiAZ()); err != nil {
		return err
	}

	if err := resourceData.Set("cloud_region", cluster.Region().ID()); err != nil {
		return err
	}

	if err := resourceData.Set("api_url", cluster.API().URL()); err != nil {
		return err
	}

	if err := resourceData.Set("console_url", cluster.Console().URL()); err != nil {
		return err
	}

	if err := resourceData.Set("compute_nodes", cluster.Nodes().Compute()); err != nil {
		return err
	}

	if err := resourceData.Set("compute_machine_type", cluster.Nodes().ComputeMachineType().ID()); err != nil {
		return err
	}

	if err := resourceData.Set("ccs_enabled", cluster.CCS().Enabled()); err != nil {
		return err
	}

	awsAccountID, ok := cluster.AWS().GetAccountID()
	if ok {
		if err := resourceData.Set("aws_account_id", awsAccountID); err != nil {
			return err
		}
	}

	awsAccessKeyID, ok := cluster.AWS().GetAccessKeyID()
	if ok {
		if err := resourceData.Set("aws_access_key_id", awsAccessKeyID); err != nil {
			return err
		}
	}

	awsSecretAccessKey, ok := cluster.AWS().GetSecretAccessKey()
	if ok {
		if err := resourceData.Set("aws_secret_access_key", awsSecretAccessKey); err != nil {
			return err
		}
	}

	awsPrivateLink, ok := cluster.AWS().GetPrivateLink()
	if ok {
		if err := resourceData.Set("aws_private_link", awsPrivateLink); err != nil {
			return err
		}
	}

	machineCIDR, ok := cluster.Network().GetMachineCIDR()
	if ok {
		if err := resourceData.Set("machine_cidr", machineCIDR); err != nil {
			return err
		}
	}

	serviceCIDR, ok := cluster.Network().GetServiceCIDR()
	if ok {
		if err := resourceData.Set("service_cidr", serviceCIDR); err != nil {
			return err
		}
	}

	podCIDR, ok := cluster.Network().GetPodCIDR()
	if ok {
		if err := resourceData.Set("pod_cidr", podCIDR); err != nil {
			return err
		}
	}

	hostPrefix, ok := cluster.Network().GetHostPrefix()
	if ok {
		if err := resourceData.Set("host_prefix", hostPrefix); err != nil {
			return err
		}
	}

	version, ok := cluster.Version().GetID()
	if ok {
		if err := resourceData.Set("version", version); err != nil {
			return err
		}
	}

	if err := resourceData.Set("state", cluster.State()); err != nil {
		return err
	}

	////////////////////

	proxy, ok := cluster.GetProxy()
	if ok {
		var tfList []interface{}
		// proxy.HTTPProxy()
		// proxy.HTTPSProxy()
		if err := resourceData.Set("proxy", cluster.Nodes().Compute()); err != nil {
			return err
		}
	}
	if err := resourceData.Set("availability_zones", cluster.Nodes().Compute()); err != nil {
		return err
	}

	if err := resourceData.Set("aws_subnet_ids", cluster.Nodes().Compute()); err != nil {
		return err
	}

	if err := resourceData.Set("properties", cluster.MultiAZ()); err != nil {
		return err
	}

	state.Properties = types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for k, v := range cluster.Properties() {
		state.Properties.Elems[k] = types.String{
			Value: v,
		}
	}
	azs, ok := cluster.Nodes().GetAvailabilityZones()
	if ok {
		state.AvailabilityZones.Elems = make([]attr.Value, 0)
		for _, az := range azs {
			state.AvailabilityZones.Elems = append(state.AvailabilityZones.Elems, types.String{
				Value: az,
			})
		}
	}
	////////////////////

	//The API does not return account id

	if ok {
		state.AWSSecretAccessKey = types.String{
			Value: awsSecretAccessKey,
		}
	} else {
		state.AWSSecretAccessKey = types.String{
			Null: true,
		}
	}

	subnetIds, ok := cluster.AWS().GetSubnetIDs()
	if ok {
		state.AWSSubnetIDs.Elems = make([]attr.Value, 0)
		for _, subnetId := range subnetIds {
			state.AWSSubnetIDs.Elems = append(state.AWSSubnetIDs.Elems, types.String{
				Value: subnetId,
			})
		}
	}

	proxy, ok := cluster.GetProxy()
	if ok {
		state.Proxy.HttpProxy = types.String{
			Value: ,
		}
		state.Proxy.HttpsProxy = types.String{
			Value: proxy.HTTPSProxy(),
		}
	}

}
