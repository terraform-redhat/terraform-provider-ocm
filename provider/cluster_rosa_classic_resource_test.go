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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/openshift-online/terraform-provider-ocm/provider/clusterresource"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	. "github.com/onsi/ginkgo/v2/dsl/core" // nolint
	. "github.com/onsi/gomega"             // nolint
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type MockHttpClient struct {
	response *http.Response
}

func (c MockHttpClient) Get(url string) (resp *http.Response, err error) {
	return c.response, nil
}

const (
	clusterId         = "1n2j3k4l5m6n7o8p9q0r"
	clusterName       = "my-cluster"
	regionId          = "us-east-1"
	multiAz           = true
	rosaCreatorArn    = "arn:aws:iam::123456789012:dummy/dummy"
	apiUrl            = "https://api.my-cluster.com:6443"
	consoleUrl        = "https://console.my-cluster.com"
	machineType       = "m5.xlarge"
	availabilityZone1 = "us-east-1a"
	availabilityZone2 = "us-east-1b"
	ccsEnabled        = true
	awsAccountID      = "123456789012"
	privateLink       = false
	oidcEndpointUrl   = "example.com"
	roleArn           = "arn:aws:iam::123456789012:role/role-name"
	httpProxy         = "http://proxy.com"
	httpsProxy        = "https://proxy.com"
)

var (
	mockHttpClient = MockHttpClient{
		response: &http.Response{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					&x509.Certificate{
						Raw: []byte("nonce"),
					},
				},
			},
		},
	}
)

func generateBasicRosaClassicClusterJson() map[string]interface{} {
	return map[string]interface{}{
		"id": clusterId,
		"region": map[string]interface{}{
			"id": regionId,
		},
		"multi_az": multiAz,
		"properties": map[string]interface{}{
			"rosa_creator_arn": rosaCreatorArn,
		},
		"api": map[string]interface{}{
			"url": apiUrl,
		},
		"console": map[string]interface{}{
			"url": consoleUrl,
		},
		"nodes": map[string]interface{}{
			"compute_machine_type": map[string]interface{}{
				"id": machineType,
			},
			"availability_zones": []interface{}{
				availabilityZone1,
			},
		},
		"ccs": map[string]interface{}{
			"enabled": ccsEnabled,
		},
		"aws": map[string]interface{}{
			"account_id":   awsAccountID,
			"private_link": privateLink,
			"sts": map[string]interface{}{
				"oidc_endpoint_url": oidcEndpointUrl,
				"role_arn":          roleArn,
			},
		},
	}
}

func generateBasicRosaClassicClusterState() *ClusterRosaClassicState {
	return &ClusterRosaClassicState{
		Name: types.String{
			Value: clusterName,
		},
		CloudRegion: types.String{
			Value: regionId,
		},
		AWSAccountID: types.String{
			Value: awsAccountID,
		},
		AvailabilityZones: types.List{
			Elems: []attr.Value{
				types.String{
					Value: availabilityZone1,
				},
				types.String{
					Value: availabilityZone2,
				},
			},
		},
		Properties: types.Map{
			Elems: map[string]attr.Value{
				"rosa_creator_arn": types.String{
					Value: rosaCreatorArn,
				},
			},
		},
		Version: types.String{
			Value: "4.10",
		},
		Proxy: &clusterresource.Proxy{
			HttpProxy: types.String{
				Value: httpProxy,
			},
			HttpsProxy: types.String{
				Value: httpsProxy,
			},
		},
		Sts: &Sts{},
	}
}

var _ = Describe("Rosa Classic Sts cluster", func() {
	Context("createClassicClusterObject", func() {
		It("Creates a cluster with correct field values", func() {
			clusterState := generateBasicRosaClassicClusterState()
			rosaClusterObject, err := createClassicClusterObject(context.Background(), clusterState, &logging.StdLogger{}, diag.Diagnostics{})
			Expect(err).To(BeNil())

			Expect(rosaClusterObject.Name()).To(Equal(clusterName))

			id, ok := rosaClusterObject.Region().GetID()
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal(regionId))

			Expect(rosaClusterObject.AWS().AccountID()).To(Equal(awsAccountID))

			availabilityZones := rosaClusterObject.Nodes().AvailabilityZones()
			Expect(availabilityZones).To(HaveLen(2))
			Expect(availabilityZones[0]).To(Equal(availabilityZone1))
			Expect(availabilityZones[1]).To(Equal(availabilityZone2))

			Expect(rosaClusterObject.Proxy().HTTPProxy()).To(Equal(httpProxy))
			Expect(rosaClusterObject.Proxy().HTTPSProxy()).To(Equal(httpsProxy))

			arn, ok := rosaClusterObject.Properties()["rosa_creator_arn"]
			Expect(ok).To(BeTrue())
			Expect(arn).To(Equal(rosaCreatorArn))
		})
	})

	It("Throws an error when version format is invalid", func() {
		clusterState := generateBasicRosaClassicClusterState()
		clusterState.Version.Value = "a.4.1"
		_, err := createClassicClusterObject(context.Background(), clusterState, &logging.StdLogger{}, diag.Diagnostics{})
		Expect(err).ToNot(BeNil())
	})

	It("Throws an error when version is unsupported", func() {
		clusterState := generateBasicRosaClassicClusterState()
		clusterState.Version.Value = "4.1.0"
		_, err := createClassicClusterObject(context.Background(), clusterState, &logging.StdLogger{}, diag.Diagnostics{})
		Expect(err).ToNot(BeNil())
	})

	Context("populateRosaClassicClusterState", func() {
		It("Converts correctly a Cluster object into a ClusterRosaClassicState", func() {
			clusterState := &ClusterRosaClassicState{}
			clusterJson := generateBasicRosaClassicClusterJson()
			clusterJsonString, err := json.Marshal(clusterJson)
			Expect(err).To(BeNil())

			clusterObject, err := cmv1.UnmarshalCluster(clusterJsonString)
			Expect(err).To(BeNil())

			populateRosaClassicClusterState(context.Background(), clusterObject, clusterState, &logging.StdLogger{}, mockHttpClient)

			Expect(clusterState.ID.Value).To(Equal(clusterId))
			Expect(clusterState.CloudRegion.Value).To(Equal(regionId))
			Expect(clusterState.MultiAZ.Value).To(Equal(multiAz))
			Expect(clusterState.Properties.Elems["rosa_creator_arn"].Equal(types.String{Value: rosaCreatorArn})).To(Equal(true))
			Expect(clusterState.APIURL.Value).To(Equal(apiUrl))
			Expect(clusterState.ConsoleURL.Value).To(Equal(consoleUrl))
			Expect(clusterState.ComputeMachineType.Value).To(Equal(machineType))
			Expect(clusterState.AvailabilityZones.Elems).To(HaveLen(1))
			Expect(clusterState.AvailabilityZones.Elems[0].Equal(types.String{Value: availabilityZone1})).To(Equal(true))
			Expect(clusterState.CCSEnabled.Value).To(Equal(ccsEnabled))
			Expect(clusterState.AWSAccountID.Value).To(Equal(awsAccountID))
			Expect(clusterState.AWSPrivateLink.Value).To(Equal(privateLink))
			Expect(clusterState.Sts.OIDCEndpointURL.Value).To(Equal(oidcEndpointUrl))
			Expect(clusterState.Sts.RoleARN.Value).To(Equal(roleArn))
		})

		It("Check trimming of oidc url with https perfix", func() {
			clusterState := &ClusterRosaClassicState{}
			clusterJson := generateBasicRosaClassicClusterJson()
			clusterJson["aws"].(map[string]interface{})["sts"].(map[string]interface{})["oidc_endpoint_url"] = "https://nonce.com"
			clusterJson["aws"].(map[string]interface{})["sts"].(map[string]interface{})["operator_role_prefix"] = "terraform-operator"

			clusterJsonString, err := json.Marshal(clusterJson)
			Expect(err).To(BeNil())
			print(string(clusterJsonString))

			clusterObject, err := cmv1.UnmarshalCluster(clusterJsonString)
			Expect(err).To(BeNil())

			err = populateRosaClassicClusterState(context.Background(), clusterObject, clusterState, &logging.StdLogger{}, mockHttpClient)
			Expect(err).To(BeNil())
			Expect(clusterState.Sts.OIDCEndpointURL.Value).To(Equal("nonce.com"))
		})

		It("Throws an error when oidc_endpoint_url is an invalid url", func() {
			clusterState := &ClusterRosaClassicState{}
			clusterJson := generateBasicRosaClassicClusterJson()
			clusterJson["aws"].(map[string]interface{})["sts"].(map[string]interface{})["oidc_endpoint_url"] = "invalid$url"
			clusterJsonString, err := json.Marshal(clusterJson)
			Expect(err).To(BeNil())
			print(string(clusterJsonString))

			clusterObject, err := cmv1.UnmarshalCluster(clusterJsonString)
			Expect(err).To(BeNil())

			err = populateRosaClassicClusterState(context.Background(), clusterObject, clusterState, &logging.StdLogger{}, mockHttpClient)
			Expect(err).To(BeNil())
			Expect(clusterState.Sts.Thumbprint.Value).To(Equal(""))
		})
	})
})
