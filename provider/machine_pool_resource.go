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
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type MachinePoolResourceType struct {
}

type MachinePoolResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *MachinePoolResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "Machine pool.",
		Attributes: map[string]tfsdk.Attribute{
			"cluster": {
				Description: "Identifier of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"id": {
				Description: "Unique identifier of the machine pool.",
				Type:        types.StringType,
				Computed:    true,
			},
			"name": {
				Description: "Name of the machine pool.",
				Type:        types.StringType,
				Required:    true,
			},
			"machine_type": {
				Description: "Identifier of the machine type used by the nodes, " +
					"for example `r5.xlarge`. Use the `ocm_machine_types` data " +
					"source to find the possible values.",
				Type:     types.StringType,
				Required: true,
			},
			"replicas": {
				Description: "The number of machines of the pool",
				Type:        types.Int64Type,
				Required:    true,
			},
		},
	}
	return
}

func (t *MachinePoolResourceType) NewResource(ctx context.Context,
	p provider.Provider) (result resource.Resource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation: use it directly when needed.
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &MachinePoolResource{
		logger:     parent.logger,
		collection: collection,
	}

	return
}

func (r *MachinePoolResource) Create(ctx context.Context,
	request resource.CreateRequest, response *resource.CreateResponse) {
	// Get the plan:
	state := &MachinePoolState{}
	diags := request.Plan.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Wait till the cluster is ready:
	resource := r.collection.Cluster(state.Cluster.Value)
	pollCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()
	_, err := resource.Poll().
		Interval(30 * time.Second).
		Predicate(func(get *cmv1.ClusterGetResponse) bool {
			return get.Body().State() == cmv1.ClusterStateReady
		}).
		StartContext(pollCtx)
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

	// Create the machine pool:
	builder := cmv1.NewMachinePool()
	builder.ID(state.Name.Value)
	builder.InstanceType(state.MachineType.Value)
	builder.Replicas(int(state.Replicas.Value))
	object, err := builder.Build()
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build machine pool",
			fmt.Sprintf(
				"Can't build machine pool for cluster '%s': %v",
				state.Cluster.Value, err,
			),
		)
		return
	}
	collection := resource.MachinePools()
	add, err := collection.Add().Body(object).SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't create machine pool",
			fmt.Sprintf(
				"Can't create machine pool for cluster '%s': %v",
				state.Cluster.Value, err,
			),
		)
		return
	}
	object = add.Body()

	// Save the state:
	r.populateState(object, state)
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *MachinePoolResource) Read(ctx context.Context, request resource.ReadRequest,
	response *resource.ReadResponse) {
	// Get the current state:
	state := &MachinePoolState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Find the identity provider:
	resource := r.collection.Cluster(state.Cluster.Value).
		MachinePools().
		MachinePool(state.ID.Value)
	get, err := resource.Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find machine pool",
			fmt.Sprintf(
				"Can't find machine pool with identifier '%s' for "+
					"cluster '%s': %v",
				state.ID.Value, state.Cluster.Value, err,
			),
		)
		return
	}
	object := get.Body()

	// Save the state:
	r.populateState(object, state)
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}

func (r *MachinePoolResource) Update(ctx context.Context, request resource.UpdateRequest,
	response *resource.UpdateResponse) {
}

func (r *MachinePoolResource) Delete(ctx context.Context, request resource.DeleteRequest,
	response *resource.DeleteResponse) {
	// Get the state:
	state := &MachinePoolState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send the request to delete the machine pool:
	resource := r.collection.Cluster(state.Cluster.Value).
		MachinePools().
		MachinePool(state.ID.Value)
	_, err := resource.Delete().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't delete machine pool",
			fmt.Sprintf(
				"Can't delete machine pool with identifier '%s' for "+
					"cluster '%s': %v",
				state.ID.Value, state.Cluster.Value, err,
			),
		)
		return
	}

	// Remove the state:
	response.State.RemoveResource(ctx)
}

func (r *MachinePoolResource) ImportState(ctx context.Context, request resource.ImportStateRequest,
	response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(
		ctx,
		tftypes.NewAttributePath().WithAttributeName("id"),
		request,
		response,
	)
}

// populateState copies the data from the API object to the Terraform state.
func (r *MachinePoolResource) populateState(object *cmv1.MachinePool, state *MachinePoolState) {
	state.ID = types.String{
		Value: object.ID(),
	}
	state.Name = types.String{
		Value: object.ID(),
	}
	state.MachineType = types.String{
		Value: object.InstanceType(),
	}
	state.Replicas = types.Int64{
		Value: int64(object.Replicas()),
	}
}
