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
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"             // nolint
	. "github.com/onsi/gomega"                         // nolint
	. "github.com/onsi/gomega/ghttp"                   // nolint
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Cluster creation", func() {
	// This is the cluster that will be returned by the server when asked to create or retrieve
	// a cluster.
	const template = `{
	  "id": "123",
	  "product": {
		"id": "rosa"
	  },
	  "name": "my-cluster",
	  "cloud_provider": {
	    "id": "aws"
	  },
	  "region": {
	    "id": "us-west-1"
	  },
	  "multi_az": true,
	  "api": {
	    "url": "https://my-api.example.com"
	  },
	  "console": {
	    "url": "https://my-console.example.com"
	  },
	  "nodes": {
	    "compute": 3,
	    "compute_machine_type": {
	      "id": "r5.xlarge"
	    }
	  },
	  "ccs": {
	    "enabled": false
	  },
	  "network": {
	    "machine_cidr": "10.0.0.0/16",
	    "service_cidr": "172.30.0.0/16",
	    "pod_cidr": "10.128.0.0/14",
	    "host_prefix": 23
	  },
	  "version": {
		  "id": "openshift-4.8.0"
	  },
	  "state": "ready"
	}`

	It("Creates basic cluster", func() {
		// Prepare the server:
		server.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters"),
				VerifyJQ(`.name`, "my-cluster"),
				VerifyJQ(`.cloud_provider.id`, "aws"),
				VerifyJQ(`.region.id`, "us-west-1"),
				VerifyJQ(`.product.id`, "rosa"),
				RespondWithJSON(http.StatusCreated, template),
			),
		)

		// Run the apply command:
		terraform.Source(`
		  resource "ocm_cluster" "my_cluster" {
		    name           = "my-cluster"
			product		   = "rosa"
		    cloud_provider = "aws"			
		    cloud_region   = "us-west-1"
		  }
		`)
		Expect(terraform.Apply()).To(BeZero())
	})

	It("Creates cluster with http proxy", func() {
		// Prepare the server:
		server.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters"),
				VerifyJQ(`.name`, "my-cluster"),
				VerifyJQ(`.cloud_provider.id`, "aws"),
				VerifyJQ(`.region.id`, "us-west-1"),
				VerifyJQ(`.product.id`, "rosa"),
				VerifyJQ(`.proxy.http_proxy`, "http://proxy.com"),
				VerifyJQ(`.proxy.https_proxy`, "http://proxy.com"),
				RespondWithPatchedJSON(http.StatusOK, template, `[
					{
					  "op": "add",
					  "path": "/proxy",
					  "value": {						  
						  "http_proxy" : "http://proxy.com",
						  "https_proxy" : "http://proxy.com"
					  }
					}]`),
			),
		)

		// Run the apply command:
		terraform.Source(`
		  resource "ocm_cluster" "my_cluster" {
		    name           = "my-cluster"
			product		   = "rosa"
		    cloud_provider = "aws"			
		    cloud_region   = "us-west-1"
			proxy = {
				http_proxy = "http://proxy.com",
				https_proxy = "http://proxy.com",
			}			
		  }
		`)
		Expect(terraform.Apply()).To(BeZero())
	})

	It("Creates cluster with aws subnet ids & private link", func() {
		// Prepare the server:
		server.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters"),
				VerifyJQ(`.name`, "my-cluster"),
				VerifyJQ(`.cloud_provider.id`, "aws"),
				VerifyJQ(`.region.id`, "us-west-1"),
				VerifyJQ(`.product.id`, "rosa"),
				VerifyJQ(`.aws.subnet_ids.[0]`, "id1"),
				VerifyJQ(`.aws.private_link`, true),
				VerifyJQ(`.nodes.availability_zones.[0]`, "az1"),
				VerifyJQ(`.api.listening`, "internal"),
				RespondWithPatchedJSON(http.StatusOK, template, `[
					{
						"op": "add",
						"path": "/availability_zones",
						"value": ["az1", "az2", "az3"]
					},					
					{
					  "op": "add",
					  "path": "/aws",
					  "value": {
						  "private_link": true,
						  "subnet_ids": ["id1", "id2", "id3"]
					  }
					}]`),
			),
		)

		// Run the apply command:
		terraform.Source(`
		  resource "ocm_cluster" "my_cluster" {
		    name           = "my-cluster"
			product		   = "rosa"
		    cloud_provider = "aws"			
		    cloud_region   = "us-west-1"
			availability_zones = ["az1","az2","az3"]
			aws_private_link = true
			aws_subnet_ids = [
				"id1", "id2", "id3"
			]
		  }
		`)
		Expect(terraform.Apply()).To(BeZero())
	})

	It("Creates cluster when private link is false", func() {
		// Prepare the server:
		server.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters"),
				VerifyJQ(`.name`, "my-cluster"),
				VerifyJQ(`.cloud_provider.id`, "aws"),
				VerifyJQ(`.region.id`, "us-west-1"),
				VerifyJQ(`.product.id`, "rosa"),
				VerifyJQ(`.aws.private_link`, false),
				VerifyJQ(`.api.listening`, nil),
				RespondWithPatchedJSON(http.StatusOK, template, `[
					{
					  "op": "add",
					  "path": "/aws",
					  "value": {
						  "private_link": false
					  }
					}]`),
			),
		)

		// Run the apply command:
		terraform.Source(`
		  resource "ocm_cluster" "my_cluster" {
		    name           = "my-cluster"
			product		   = "rosa"
		    cloud_provider = "aws"			
		    cloud_region   = "us-west-1"
			aws_private_link = false
		  }
		`)
		Expect(terraform.Apply()).To(BeZero())
	})

	It("Creates rosa sts cluster", func() {
		// Prepare the server:
		server.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters"),
				VerifyJQ(`.name`, "my-cluster"),
				VerifyJQ(`.cloud_provider.id`, "aws"),
				VerifyJQ(`.region.id`, "us-west-1"),
				VerifyJQ(`.product.id`, "rosa"),
				VerifyJQ(`.aws.sts.role_arn`, "arn:aws:iam::account-id:role/ManagedOpenShift-Installer-Role"),
				VerifyJQ(`.aws.sts.support_role_arn`, "arn:aws:iam::account-id:role/ManagedOpenShift-Support-Role"),
				VerifyJQ(`.aws.sts.instance_iam_roles.master_role_arn`, "arn:aws:iam::account-id:role/ManagedOpenShift-ControlPlane-Role"),
				VerifyJQ(`.aws.sts.instance_iam_roles.worker_role_arn`, "arn:aws:iam::account-id:role/ManagedOpenShift-Worker-Role"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[0].role_arn`, "arn:aws:iam::account-id:role/cloud-credential"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[1].role_arn`, "arn:aws:iam::account-id:role/image-registry"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[2].role_arn`, "arn:aws:iam::account-id:role/ingress"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[3].role_arn`, "arn:aws:iam::account-id:role/ebs"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[4].role_arn`, "arn:aws:iam::account-id:role/cloud-network-config"),
				VerifyJQ(`.aws.sts.operator_iam_roles.[5].role_arn`, "arn:aws:iam::account-id:role/machine-api"),
				RespondWithPatchedJSON(http.StatusOK, template, `[
					{
					  "op": "add",
					  "path": "/aws",
					  "value": {
						  "sts" : {
							  "oidc_endpoint_url": "https://oidc_endpoint_url",
							  "role_arn": "arn:aws:iam::account-id:role/ManagedOpenShift-Installer-Role",
							  "support_role_arn": "arn:aws:iam::account-id:role/ManagedOpenShift-Support-Role",
							  "instance_iam_roles" : {
								"master_role_arn" : "arn:aws:iam::account-id:role/ManagedOpenShift-ControlPlane-Role",
								"worker_role_arn" : "arn:aws:iam::account-id:role/ManagedOpenShift-Worker-Role"
							  },
							  "operator_iam_roles" : [
								{
									"name": "cloud-credential-operator-iam-ro-creds",
									"namespace": "openshift-cloud-credential-operator",
									"role_arn": "arn:aws:iam::account-id:role/cloud-credential"
								  },
								  {
									"name": "installer-cloud-credentials",
									"namespace": "openshift-image-registry",
									"role_arn": "arn:aws:iam::account-id:role/image-registry"
								  },
								  {
									"name": "cloud-credentials",
									"namespace": "openshift-ingress-operator",
									"role_arn": "arn:aws:iam::account-id:role/ingress"
								  },
								  {
									"name": "ebs-cloud-credentials",
									"namespace": "openshift-cluster-csi-drivers",
									"role_arn": "arn:aws:iam::account-id:role/ebs"
								  },
								  {
									"name": "cloud-credentials",
									"namespace": "openshift-cloud-network-config-controller",
									"role_arn": "arn:aws:iam::account-id:role/cloud-network-config"
								  },
								  {
									"name": "aws-cloud-credentials",
									"namespace": "openshift-machine-api",
									"role_arn": "arn:aws:iam::account-id:role/machine-api"
								  }								  
							  ]
						  }
					  }
					}
				  ]`),
			),
		)

		// Run the apply command:
		terraform.Source(`
		resource "ocm_cluster" "my_cluster" {
			name           = "my-cluster"
			product		   = "rosa"
			cloud_provider = "aws"			
			cloud_region   = "us-west-1"
			sts = {
				role_arn = "arn:aws:iam::account-id:role/ManagedOpenShift-Installer-Role",
				support_role_arn = "arn:aws:iam::account-id:role/ManagedOpenShift-Support-Role",
				operator_iam_roles = [
					{
						name =  "cloud-credential-operator-iam-ro-creds",
						namespace = "openshift-cloud-credential-operator",
						role_arn = "arn:aws:iam::account-id:role/cloud-credential",
					},
					{
						name =  "installer-cloud-credentials",
						namespace = "openshift-image-registry",
						role_arn = "arn:aws:iam::account-id:role/image-registry",
					},
					{
						name =  "cloud-credentials",
						namespace = "openshift-ingress-operator",
						role_arn = "arn:aws:iam::account-id:role/ingress",
					},
					{
						name =  "ebs-cloud-credentials",
						namespace = "openshift-cluster-csi-drivers",
						role_arn = "arn:aws:iam::account-id:role/ebs",
					},
					{
						name =  "cloud-credentials",
						namespace = "openshift-cloud-network-config-controller",
						role_arn = "arn:aws:iam::account-id:role/cloud-network-config",
					},
					{
						name =  "aws-cloud-credentials",
						namespace = "openshift-machine-api",
						role_arn = "arn:aws:iam::account-id:role/machine-api",
					},
				]
				instance_iam_roles = {
				  master_role_arn = "arn:aws:iam::account-id:role/ManagedOpenShift-ControlPlane-Role",
				  worker_role_arn = "arn:aws:iam::account-id:role/ManagedOpenShift-Worker-Role"
				},
			}
		  }
		`)
		Expect(terraform.Apply()).To(BeZero())
	})
})
