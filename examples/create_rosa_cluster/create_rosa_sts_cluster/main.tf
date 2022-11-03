#
# Copyright (c) 2022 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.20.0"
    }
    ocm = {
      version = ">= 0.1"
      source  = "openshift-online/ocm"
    }
  }
}

variable token {
  type = string
  sensitive = true
}

variable operator_role_prefix {
    type = string
}

provider "ocm" {
  token = var.token
}

locals {
  sts_roles = {
      role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Installer-Role",
      support_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Support-Role",
      instance_iam_roles = {
              master_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-ControlPlane-Role",
              worker_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Worker-Role"
      },

      operator_iam_roles = [
        {
          name =  "cloud-credential-operator-iam-ro-creds",
          namespace = "openshift-cloud-credential-operator",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-cloud-credential-operator-cloud-c",
        },
        {
          name =  "installer-cloud-credentials",
          namespace = "openshift-image-registry",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-image-registry-installer-cloud-cr",
        },
        {
          name =  "cloud-credentials",
          namespace = "openshift-ingress-operator",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-ingress-operator-cloud-credential",
        },
        {
          name =  "ebs-cloud-credentials",
          namespace = "openshift-cluster-csi-drivers",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-cluster-csi-drivers-ebs-cloud-cre",
        },
        {
          name =  "cloud-credentials",
          namespace = "openshift-cloud-network-config-controller",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-cloud-network-config-controller-c",
        },
        {
          name =  "aws-cloud-credentials",
          namespace = "openshift-machine-api",
          role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.operator_role_prefix}-openshift-machine-api-aws-cloud-credentials",
        },
      ]
  }
}

data "aws_caller_identity" "current" {
}

resource "ocm_cluster_rosa_classic" "rosa_sts_cluster" {
  name           = "my-cluster"
  cloud_region   = "us-east-2"
  aws_account_id     = data.aws_caller_identity.current.account_id
  availability_zones = ["us-east-2a"]
  properties = {
    rosa_creator_arn = data.aws_caller_identity.current.arn
  }
  wait = false
  sts = local.sts_roles
}

module operator_roles {
    source  = "https://github.com/openshift-online/terraform-provider-ocm/modules/ocm_cluster_rosa_classic/operator_roles"
    cluster_id = ocm_cluster_rosa_classic.rosa_sts_cluster.id
    operator_role_prefix = var.operator_role_prefix
    rh_oidc_provider_thumbprint = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.thumbprint
    rh_oidc_provider_url = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.oidc_endpoint_url

}
