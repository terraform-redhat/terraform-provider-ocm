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

type ClusterState struct {
	APIURL             string
	CloudProvider      string
	CloudRegion        string
	ConsoleURL         string
	ID                 string
	Product            string
	State              string
	Name               string
	AWSAccessKeyID     *string
	AWSAccountID       *string
	AWSSecretAccessKey *string
	AWSSubnetIDs       *[]string
	AWSPrivateLink     *bool
	CCSEnabled         *bool
	ComputeMachineType *string
	ComputeNodes       *int64
	HostPrefix         *int64
	MachineCIDR        *string
	MultiAZ            *bool
	AvailabilityZones  *[]string
	PodCIDR            *string
	Properties         *map[string]string
	ServiceCIDR        *string
	Proxy              *Proxy
	Version            *string
	Wait               *bool
}

type Proxy struct {
	HttpProxy  string
	HttpsProxy string
	NoProxy    string
}
