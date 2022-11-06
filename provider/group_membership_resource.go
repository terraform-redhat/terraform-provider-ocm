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

type GroupMembershipResourceType struct {
}

type GroupMembershipResource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *GroupMembershipResourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "Manages user group membership.",
		Attributes: map[string]tfsdk.Attribute{
			"cluster": {
				Description: "Identifier of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"group": {
				Description: "Identifier of the group.",
				Type:        types.StringType,
				Required:    true,
			},
			"id": {
				Description: "Identifier of the membership.",
				Type:        types.StringType,
				Computed:    true,
			},
			"user": {
				Description: "user name.",
				Type:        types.StringType,
				Required:    true,
			},
		},
	}
	return
}

func (t *GroupMembershipResourceType) NewResource(ctx context.Context,
	p provider.Provider) (result resource.Resource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation: use it directly when needed.
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &GroupMembershipResource{
		logger:     parent.logger,
		collection: collection,
	}

	return
}

func (r *GroupMembershipResource) Create(ctx context.Context,
	request resource.CreateRequest, response *resource.CreateResponse) {
	// Get the plan:
	state := &GroupMembershipState{}
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

	// Create the membership:
	builder := cmv1.NewUser()
	builder.ID(state.User.Value)
	object, err := builder.Build()
	if err != nil {
		response.Diagnostics.AddError(
			"Can't build group membership",
			fmt.Sprintf(
				"Can't build group membership for cluster '%s' and group '%s': %v",
				state.Cluster.Value, state.Group.Value, err,
			),
		)
		return
	}
	collection := resource.Groups().Group(state.Group.Value).Users()
	add, err := collection.Add().Body(object).SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't create group membership",
			fmt.Sprintf(
				"Can't create group membership for cluster '%s' and group '%s': %v",
				state.Cluster.Value, state.Group.Value, err,
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

func (r *GroupMembershipResource) Read(ctx context.Context, request resource.CreateRequest,
	response *resource.CreateResponse) {
	// Get the current state:
	state := &GroupMembershipState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Find the group membership:
	resource := r.collection.Cluster(state.Cluster.Value).Groups().Group(state.Group.Value).
		Users().
		User(state.ID.Value)
	get, err := resource.Get().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't find group membership",
			fmt.Sprintf(
				"Can't find user group membership identifier '%s' for "+
					"cluster '%s' and group '%s': %v",
				state.ID.Value, state.Cluster.Value, state.Group.Value, err,
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

func (r *GroupMembershipResource) Update(ctx context.Context, request resource.UpdateRequest,
	response *resource.UpdateResponse) {
}

func (r *GroupMembershipResource) Delete(ctx context.Context, request resource.DeleteRequest,
	response *resource.DeleteResponse) {
	// Get the state:
	state := &GroupMembershipState{}
	diags := request.State.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Send the request to delete group membership:
	resource := r.collection.Cluster(state.Cluster.Value).Groups().Group(state.Group.Value).
		Users().
		User(state.ID.Value)
	_, err := resource.Delete().SendContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(
			"Can't delete group membership",
			fmt.Sprintf(
				"Can't delete group membership with identifier '%s' for "+
					"cluster '%s' and group '%s': %v",
				state.ID.Value, state.Cluster.Value, state.Group.Value, err,
			),
		)
		return
	}

	// Remove the state:
	response.State.RemoveResource(ctx)
}

func (r *GroupMembershipResource) ImportState(ctx context.Context, request resource.ImportStateRequest,
	response *resource.ImportStateResponse) {
	resource.ResourceImportStatePassthroughID(
		ctx,
		tftypes.NewAttributePath().WithAttributeName("id"),
		request,
		response,
	)
}

// populateState copies the data from the API object to the Terraform state.
func (r *GroupMembershipResource) populateState(object *cmv1.User, state *GroupMembershipState) {
	state.ID = types.String{
		Value: object.ID(),
	}
	state.User = types.String{
		Value: object.ID(),
	}
}
