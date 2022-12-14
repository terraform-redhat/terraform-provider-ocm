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

package cloudprovider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/terraform-provider-ocm/provider/util"
)

func DataSourceCloudProvider() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCloudProviderRead,
		Schema: map[string]*schema.Schema{
			"search": {
				Description: "Search criteria.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"order": {
				Description: "Order criteria.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"item": {
				Type:        schema.TypeMap,
				Description: "Content of the list when there is exactly one item.",
				Computed:    true,
				Elem:        itemSchema(),
			},
			"items": {
				Description: "Content of the list.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        itemSchema(),
			},
		},
	}
}

func itemSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "Unique identifier of the cloud provider. This is what " +
					"should be used when referencing the cloud provider from other " +
					"places, for example in the 'cloud_provider' attribute " +
					"of the cluster resource.",
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Description: "Short name of the cloud provider, for example 'aws' " +
					"or 'gcp'.",
				Type:     schema.TypeString,
				Computed: true,
			},
			"display_name": {
				Description: "Human friendly name of the cloud provider, for example " +
					"'AWS' or 'GCP'",
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudProviderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connection := meta.(*sdk.Connection)
	util.Logger.Info(ctx, "[INFO] Reading cloud providers")

	// Fetch the complete list of cloud providers:
	var listItems []*cmv1.CloudProvider
	listSize := 100
	listPage := 1
	listRequest := connection.ClustersMgmt().V1().CloudProviders().List().Size(listSize)

	state := &CloudProvidersState{}
	search, ok := d.GetOk("search")
	if ok {
		state.Search = search.(string)
		listRequest.Search(state.Search)
	}

	order, ok := d.GetOk("order")
	if ok {
		state.Order = order.(string)
		listRequest.Order(state.Order)
	}

	for {
		listResponse, err := listRequest.SendContext(ctx)
		if err != nil {
			return diag.Errorf("Can't list cloud providers",
				err.Error())
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

	return nil
}
