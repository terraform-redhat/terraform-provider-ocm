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
      version = " 0.0.2"
      source  = "terraform-redhat/ocm"
    }
  }
}

provider "ocm" {
  token = var.token
  url = var.url
}

locals {
  sts_roles = {
      role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.account_role_prefix}-Installer-Role",
      support_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.account_role_prefix}-Support-Role",
      instance_iam_roles = {
        master_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.account_role_prefix}-ControlPlane-Role",
        worker_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.account_role_prefix}-Worker-Role"
      },
      operator_role_prefix = var.operator_role_prefix,
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
  sts = local.sts_roles
  # disable_waiting_in_destroy = false
  # destroy_timeout in minutes
  destroy_timeout = 60
}

resource "ocm_cluster_wait" "rosa_cluster" {
  cluster = ocm_cluster_rosa_classic.rosa_sts_cluster.id
  # timeout in minutes
  timeout = 60
}

data "ocm_rosa_operator_roles" "operator_roles" {
  operator_role_prefix = var.operator_role_prefix
  account_role_prefix = var.account_role_prefix
}

module operator_roles {
  source = "terraform-redhat/rosa-sts/aws"
  version = "0.0.2"

  create_operator_roles = true
  create_oidc_provider = true
  create_account_roles = false

  cluster_id = ocm_cluster_rosa_classic.rosa_sts_cluster.id
  rh_oidc_provider_thumbprint = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.thumbprint
  rh_oidc_provider_url = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.oidc_endpoint_url
  operator_roles_properties = data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles
}

