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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type GroupsDataSourceType struct {
}

type GroupsDataSource struct {
	logger     logging.Logger
	collection *cmv1.ClustersClient
}

func (t *GroupsDataSourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "List of groups.",
		Attributes: map[string]tfsdk.Attribute{
			"cluster": {
				Description: "Identifier of the cluster.",
				Type:        types.StringType,
				Required:    true,
			},
			"items": {
				Description: "Items of the list.",
				Attributes:  t.itemSchema(),
				Computed:    true,
			},
		},
	}
	return
}

func (t *GroupsDataSourceType) itemSchema() tfsdk.NestedAttributes {
	return tfsdk.ListNestedAttributes(
		map[string]tfsdk.Attribute{
			"id": {
				Description: "Unique identifier of the group. This is what " +
					"should be used when referencing the group from other " +
					"places, for example in the 'group' attribute of the " +
					"user resource.",
				Type:     types.StringType,
				Computed: true,
			},
			"name": {
				Description: "Short name of the group for example " +
					"'dedicated-admins'.",
				Type:     types.StringType,
				Computed: true,
			},
		},
		tfsdk.ListNestedAttributesOptions{},
	)
}

func (t *GroupsDataSourceType) NewDataSource(ctx context.Context,
	p provider.Provider) (result datasource.DataSource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation:
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().Clusters()

	// Create the resource:
	result = &GroupsDataSource{
		logger:     parent.logger,
		collection: collection,
	}
	return
}

func (s *GroupsDataSource) Read(ctx context.Context, request datasource.ReadRequest,
	response *datasource.ReadResponse) {
	// Get the state:
	state := &GroupsState{}
	diags := request.Config.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Fetch the complete list of groups of the cluster:
	var listItems []*cmv1.Group
	listSize := 10
	listPage := 1
	listRequest := s.collection.Cluster(state.Cluster.Value).Groups().List()
	for {
		listResponse, err := listRequest.SendContext(ctx)
		if err != nil {
			response.Diagnostics.AddError(
				"Can't list groups",
				err.Error(),
			)
			return
		}
		if listItems == nil {
			listItems = make([]*cmv1.Group, 0, listResponse.Total())
		}
		listResponse.Items().Each(func(listItem *cmv1.Group) bool {
			listItems = append(listItems, listItem)
			return true
		})
		if listResponse.Size() < listSize {
			break
		}
		listPage++
		listRequest.Page(listPage)
	}

	// Populate the state:
	state.Items = make([]*GroupState, len(listItems))
	for i, listItem := range listItems {
		state.Items[i] = &GroupState{
			ID: types.String{
				Value: listItem.ID(),
			},
			Name: types.String{
				Value: listItem.ID(),
			},
		}
	}

	// Save the state:
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}
