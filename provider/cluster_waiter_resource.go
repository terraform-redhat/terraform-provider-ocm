package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type Waiter interface {
	Ready() bool
}

type ClusterWaiterResourceType struct {
}

type ClusterWaiterResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

const (
	defaultTimeoutInMinutes   = int64(60)
	nonPositiveTimeoutSummary = "Can't poll cluster state with a non-positive timeout"
	nonPositiveTimeoutFormat  = "Can't poll state of cluster with identifier '%s', the timeout that was set is not a positive number"
	pollingIntervalInMinutes  = 2
)

func (t *ClusterWaiterResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "Wait Cluster Resource To be Ready",
		Attributes: map[string]tfsdk.Attribute{
			"cluster": {
				Description: "Identifier of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"timeout": {
				Description: "An optional timeout till the cluster is ready. The timeout value should be in minutes." +
					" the default value is 60 minutes",
				Type:     types.Int64Type,
				Optional: true,
			},
			"ready": {
				Description: "Whether the cluster is ready",
				Type:        types.BoolType,
				Computed:    true,
			},
		},
	}
	return
}

func (t *ClusterWaiterResourceType) NewResource(ctx context.Context,
	p tfsdk.Provider) (result tfsdk.Resource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation: use it directly when needed.
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &ClusterWaiterResource{
		logger:     parent.logger,
		collection: collection,
	}
	return
}

func (r *ClusterWaiterResource) Create(ctx context.Context,
	request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	// Get the plan:
	state := &ClusterWaiterState{}
	diags := request.Plan.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	state.Ready = types.Bool{
		Value: false,
	}

	timeout := defaultTimeoutInMinutes
	if !state.Timeout.Unknown && !state.Timeout.Null {
		if state.Timeout.Value <= 0 {
			response.Diagnostics.AddWarning(nonPositiveTimeoutSummary, fmt.Sprintf(nonPositiveTimeoutFormat, state.Cluster.Value))
		} else {
			timeout = state.Timeout.Value
		}
	}

	// Wait till the cluster is ready:
	object, err := r.retryClusterReadiness(3, 30*time.Second, state.Cluster.Value, ctx, timeout)
	if err != nil {

		response.Diagnostics.AddError(
			"Can't poll cluster state",
			fmt.Sprintf(
				"Can't poll state of cluster with identifier '%s': %v",
				state.Cluster.Value, err,
			),
		)
		return
	}
	isClusterReady := false
	if object.State() == cmv1.ClusterStateReady {
		isClusterReady = true
	}

	state.Ready = types.Bool{
		Value: isClusterReady,
	}

	// Save the state:
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *ClusterWaiterResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest,
	response *tfsdk.ReadResourceResponse) {
	// Do Nothing
}

func (r *ClusterWaiterResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest,
	response *tfsdk.UpdateResourceResponse) {
	// Do Nothing
}

func (r *ClusterWaiterResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest,
	response *tfsdk.DeleteResourceResponse) {
	response.State.RemoveResource(ctx)
}

func (r *ClusterWaiterResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest,
	response *tfsdk.ImportResourceStateResponse) {
	// Do Nothing
}

func (r *ClusterWaiterResource) isClusterReady(clusterId string, ctx context.Context, timeout int64) (*cmv1.Cluster, error) {
	resource := r.collection.Cluster(clusterId)
	var object *cmv1.Cluster
	pollCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Minute)
	defer cancel()
	_, err := resource.Poll().
		Interval(pollingIntervalInMinutes * time.Minute).
		Predicate(func(getClusterResponse *cmv1.ClusterGetResponse) bool {
			object = getClusterResponse.Body()
			r.logger.Debug(ctx, "cluster state is %s", object.State())
			return object.State() == cmv1.ClusterStateReady
		}).
		StartContext(pollCtx)
	if err != nil {
		r.logger.Error(ctx, "Can't  poll cluster state")
		return nil, err
	}

	return object, err
}

func (r *ClusterWaiterResource) retryClusterReadiness(attempts int, sleep time.Duration, clusterId string, ctx context.Context, timeout int64) (*cmv1.Cluster, error) {
	object, err := r.isClusterReady(clusterId, ctx, timeout)
	if err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			return r.retryClusterReadiness(attempts, 2*sleep, clusterId, ctx, timeout)
		}
		return object, err
	}

	return object, nil
}
