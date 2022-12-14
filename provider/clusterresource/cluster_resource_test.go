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
	"encoding/json"
	. "github.com/onsi/ginkgo/v2/dsl/core" // nolint
	. "github.com/onsi/gomega"             // nolint
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Cluster creation", func() {
	clusterId := "1n2j3k4l5m6n7o8p9q0r"
	clusterName := "my-cluster"
	productId := "rosa"
	cloudProviderId := "aws"
	regionId := "us-east-1"
	multiAz := true
	rosaCreatorArn := "arn:aws:iam::123456789012:dummy/dummy"
	apiUrl := "https://api.my-cluster.com:6443"
	consoleUrl := "https://console.my-cluster.com"
	machineType := "m5.xlarge"
	availabilityZone := "us-east-1a"
	ccsEnabled := true
	awsAccountID := "123456789012"
	awsAccessKeyID := "AKIAIOSFODNN7EXAMPLE"
	awsSecretAccessKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	privateLink := false
	wait := false
	az := []string{availabilityZone}
	properties := map[string]string{
		"rosa_creator_arn": rosaCreatorArn,
	}
	It("Creates ClusterBuilder with correct field values", func() {
		clusterState := &ClusterState{
			Name:              clusterName,
			CloudRegion:       regionId,
			AWSAccountID:      &awsAccountID,
			AvailabilityZones: &az,
			Properties:        &properties,
			Wait:              &wait,
		}
		clusterObject, err := createClusterObject(context.Background(), clusterState)
		Expect(err).To(BeNil())

		Expect(err).To(BeNil())
		Expect(clusterObject).ToNot(BeNil())

		Expect(clusterObject.Name()).To(Equal(clusterName))

		id, ok := clusterObject.Region().GetID()
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal(regionId))

		Expect(clusterObject.AWS().AccountID()).To(Equal(awsAccountID))

		availabilityZones := clusterObject.Nodes().AvailabilityZones()
		Expect(availabilityZones).To(HaveLen(1))
		Expect(availabilityZones[0]).To(Equal(availabilityZone))

		arn, ok := clusterObject.Properties()["rosa_creator_arn"]
		Expect(ok).To(BeTrue())
		Expect(arn).To(Equal(arn))
	})

	It("populateClusterState converts correctly a Cluster object into a ClusterState", func() {
		// We builder a Cluster object by creating a json and using cmv1.UnmarshalCluster on it
		clusterJson := map[string]interface{}{
			"id": clusterId,
			"product": map[string]interface{}{
				"id": productId,
			},
			"cloud_provider": map[string]interface{}{
				"id": cloudProviderId,
			},
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
					availabilityZone,
				},
			},
			"ccs": map[string]interface{}{
				"enabled": ccsEnabled,
			},
			"aws": map[string]interface{}{
				"account_id":        awsAccountID,
				"access_key_id":     awsAccessKeyID,
				"secret_access_key": awsSecretAccessKey,
				"private_link":      privateLink,
			},
		}
		clusterJsonString, err := json.Marshal(clusterJson)
		Expect(err).To(BeNil())

		clusterObject, err := cmv1.UnmarshalCluster(clusterJsonString)
		Expect(err).To(BeNil())

		//We convert the Cluster object into a ClusterState and check that the conversion is correct
		clusterState := &ClusterState{}
		populateClusterState(clusterObject, clusterState)

		Expect(clusterState.ID).To(Equal(clusterId))
		Expect(clusterState.Product).To(Equal(productId))
		Expect(clusterState.CloudProvider).To(Equal(cloudProviderId))
		Expect(clusterState.CloudRegion).To(Equal(regionId))
		Expect(clusterState.MultiAZ).NotTo(BeNil())
		Expect(*clusterState.MultiAZ).To(Equal(multiAz))
		Expect(clusterState.Properties).NotTo(BeNil())
		Expect((*clusterState.Properties)["rosa_creator_arn"]).To(Equal(rosaCreatorArn))
		Expect(clusterState.APIURL).To(Equal(apiUrl))
		Expect(clusterState.ConsoleURL).To(Equal(consoleUrl))
		Expect(clusterState.ComputeMachineType).NotTo(BeNil())
		Expect(*clusterState.ComputeMachineType).To(Equal(machineType))
		Expect(clusterState.AvailabilityZones).NotTo(BeNil())
		Expect(*clusterState.AvailabilityZones).To(HaveLen(1))
		Expect((*clusterState.AvailabilityZones)[0]).To(Equal(availabilityZone))
		Expect(clusterState.CCSEnabled).NotTo(BeNil())
		Expect(*clusterState.CCSEnabled).To(Equal(ccsEnabled))
		Expect(clusterState.AWSAccountID).NotTo(BeNil())
		Expect(*clusterState.AWSAccountID).To(Equal(awsAccountID))
		Expect(clusterState.AWSAccessKeyID).NotTo(BeNil())
		Expect(*clusterState.AWSAccessKeyID).To(Equal(awsAccessKeyID))
		Expect(clusterState.AWSSecretAccessKey).NotTo(BeNil())
		Expect(*clusterState.AWSSecretAccessKey).To(Equal(awsSecretAccessKey))
		Expect(clusterState.AWSPrivateLink).NotTo(BeNil())
		Expect(*clusterState.AWSPrivateLink).To(Equal(privateLink))
	})
})
