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

type CloudProvidersDataSourceType struct {
}

type CloudProvidersDataSource struct {
	logger     logging.Logger
	collection *cmv1.CloudProvidersClient
}

func (t *CloudProvidersDataSourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "List of cloud providers.",
		Attributes: map[string]tfsdk.Attribute{
			"search": {
				Description: "Search criteria.",
				Type:        types.StringType,
				Optional:    true,
			},
			"order": {
				Description: "Order criteria.",
				Type:        types.StringType,
				Optional:    true,
			},
			"item": {
				Description: "Content of the list when there is exactly one item.",
				Attributes:  tfsdk.SingleNestedAttributes(t.itemAttributes()),
				Computed:    true,
			},
			"items": {
				Description: "Content of the list.",
				Attributes: tfsdk.ListNestedAttributes(
					t.itemAttributes(),
					tfsdk.ListNestedAttributesOptions{},
				),
				Computed: true,
			},
		},
	}
	return
}

func (t *CloudProvidersDataSourceType) itemAttributes() map[string]tfsdk.Attribute {
	return map[string]tfsdk.Attribute{
		"id": {
			Description: "Unique identifier of the cloud provider. This is what " +
				"should be used when referencing the cloud provider from other " +
				"places, for example in the 'cloud_provider' attribute " +
				"of the cluster resource.",
			Type:     types.StringType,
			Computed: true,
		},
		"name": {
			Description: "Short name of the cloud provider, for example 'aws' " +
				"or 'gcp'.",
			Type:     types.StringType,
			Computed: true,
		},
		"display_name": {
			Description: "Human friendly name of the cloud provider, for example " +
				"'AWS' or 'GCP'",
			Type:     types.StringType,
			Computed: true,
		},
	}
}

func (t *CloudProvidersDataSourceType) NewDataSource(ctx context.Context,
	p provider.Provider) (result datasource.DataSource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation:
	parent := p.(*Provider)

	// Get the collection of clusters:
	collection := parent.connection.ClustersMgmt().V1().CloudProviders()

	// Create the resource:
	result = &CloudProvidersDataSource{
		logger:     parent.logger,
		collection: collection,
	}
	return
}

func (s *CloudProvidersDataSource) Read(ctx context.Context, request datasource.ReadRequest,
	response *datasource.ReadResponse) {
	// Get the state:
	state := &CloudProvidersState{}
	diags := request.Config.Get(ctx, state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Fetch the complete list of cloud providers:
	var listItems []*cmv1.CloudProvider
	listSize := 100
	listPage := 1
	listRequest := s.collection.List().Size(listSize)
	if !state.Search.Unknown && !state.Search.Null {
		listRequest.Search(state.Search.Value)
	}
	if !state.Order.Unknown && !state.Order.Null {
		listRequest.Order(state.Order.Value)
	}
	for {
		listResponse, err := listRequest.SendContext(ctx)
		if err != nil {
			response.Diagnostics.AddError(
				"Can't list cloud providers",
				err.Error(),
			)
			return
		}
		if listItems == nil {
			listItems = make([]*cmv1.CloudProvider, 0, listResponse.Total())
		}
		listResponse.Items().Each(func(listItem *cmv1.CloudProvider) bool {
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
	state.Items = make([]*CloudProviderState, len(listItems))
	for i, listItem := range listItems {
		state.Items[i] = &CloudProviderState{
			ID:          listItem.ID(),
			Name:        listItem.Name(),
			DisplayName: listItem.DisplayName(),
		}
	}
	if len(state.Items) == 1 {
		state.Item = state.Items[0]
	} else {
		state.Item = nil
	}

	// Save the state:
	diags = response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}
