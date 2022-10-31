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
	"math"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type MachineTypesDataSourceType struct {
}

type MachineTypesDataSource struct {
	logger     logging.Logger
	collection *cmv1.MachineTypesClient
}

func (t *MachineTypesDataSourceType) GetSchema(ctx context.Context) (result tfsdk.Schema,
	diags diag.Diagnostics) {
	result = tfsdk.Schema{
		Description: "List of cloud providers.",
		Attributes: map[string]tfsdk.Attribute{
			"items": {
				Description: "Items of the list.",
				Attributes: tfsdk.ListNestedAttributes(
					map[string]tfsdk.Attribute{
						"cloud_provider": {
							Description: "Unique identifier of the " +
								"cloud provider where the machine " +
								"type is supported.",
							Type:     types.StringType,
							Computed: true,
						},
						"id": {
							Description: "Unique identifier of the " +
								"machine type.",
							Type:     types.StringType,
							Computed: true,
						},
						"name": {
							Description: "Short name of the machine " +
								"type.",
							Type:     types.StringType,
							Computed: true,
						},
						"cpu": {
							Description: "Number of CPU cores.",
							Type:        types.Int64Type,
							Computed:    true,
						},
						"ram": {
							Description: "Amount of RAM in bytes.",
							Type:        types.Int64Type,
							Computed:    true,
						},
					},
					tfsdk.ListNestedAttributesOptions{},
				),
				Computed: true,
			},
		},
	}
	return
}

func (t *MachineTypesDataSourceType) NewDataSource(ctx context.Context,
	p tfsdk.Provider) (result tfsdk.DataSource, diags diag.Diagnostics) {
	// Cast the provider interface to the specific implementation:
	parent := p.(*Provider)

	// Get the collection of machine types:
	collection := parent.Connection.ClustersMgmt().V1().MachineTypes()

	// Create the resource:
	result = &MachineTypesDataSource{
		logger:     parent.Logger,
		collection: collection,
	}
	return
}

func (s *MachineTypesDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest,
	response *tfsdk.ReadDataSourceResponse) {
	// Fetch the complete list of machine types:
	var listItems []*cmv1.MachineType
	listSize := 10
	listPage := 1
	listRequest := s.collection.List().Size(listSize)
	for {
		listResponse, err := listRequest.SendContext(ctx)
		if err != nil {
			response.Diagnostics.AddError(
				"Can't list machine types",
				err.Error(),
			)
			return
		}
		if listItems == nil {
			listItems = make([]*cmv1.MachineType, 0, listResponse.Total())
		}
		listResponse.Items().Each(func(listItem *cmv1.MachineType) bool {
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
	state := &MachineTypesState{
		Items: make([]*MachineTypeState, len(listItems)),
	}
	for i, listItem := range listItems {
		cpuObject := listItem.CPU()
		cpuValue := cpuObject.Value()
		cpuUnit := cpuObject.Unit()
		switch cpuUnit {
		case "vCPU":
			// Nothing.
		default:
			response.Diagnostics.AddError(
				"Unknown CPU unit",
				fmt.Sprintf("Don't know how to convert CPU unit '%s'", cpuUnit),
			)
			return
		}
		ramObject := listItem.Memory()
		ramValue := ramObject.Value()
		ramUnit := ramObject.Unit()
		switch strings.ToLower(ramUnit) {
		case "b":
			// Nothing.
		case "kb":
			ramValue *= math.Pow10(3)
		case "mb":
			ramValue *= math.Pow10(6)
		case "gb":
			ramValue *= math.Pow10(9)
		case "tb":
			ramValue *= math.Pow10(12)
		case "pb":
			ramValue *= math.Pow10(15)
		case "kib":
			ramValue *= math.Pow(2, 10)
		case "mib":
			ramValue *= math.Pow(2, 20)
		case "gib":
			ramValue *= math.Pow(2, 30)
		case "tib":
			ramValue *= math.Pow(2, 40)
		case "pib":
			ramValue *= math.Pow(2, 50)
		default:
			response.Diagnostics.AddError(
				"Unknown RAM unit",
				fmt.Sprintf("Don't know how to convert RAM unit '%s'", ramUnit),
			)
			return
		}
		state.Items[i] = &MachineTypeState{
			CloudProvider: listItem.CloudProvider().ID(),
			ID:            listItem.ID(),
			Name:          listItem.Name(),
			CPU:           int64(cpuValue),
			RAM:           int64(ramValue),
		}
	}

	// Save the state:
	diags := response.State.Set(ctx, state)
	response.Diagnostics.Append(diags...)
}
