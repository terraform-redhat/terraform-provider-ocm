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


provider "ocm" {
  token = var.token
  url = var.url
}

locals {
  sts_roles = {
      role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Installer-Role",
      support_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Support-Role",
      instance_iam_roles = {
        master_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-ControlPlane-Role",
        worker_role_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ManagedOpenShift-Worker-Role"
      },
      operator_role_prefix = var.operator_role_prefix,
      account_role_prefix = var.account_role_prefix,

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
}

data "ocm_rosa_operator_roles" "operator_roles" {
  cluster_id = ocm_cluster_rosa_classic.rosa_sts_cluster.id
  operator_role_prefix = var.operator_role_prefix
  account_role_prefix = var.account_role_prefix
}

module operator_roles {
    source  = "git::https://github.com/openshift-online/terraform-provider-ocm.git//modules/operator_roles"
    for_each = data.ocm_rosa_operator_roles.operator_roles

    cluster_id = ocm_cluster_rosa_classic.rosa_sts_cluster.id
    rh_oidc_provider_thumbprint = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.thumbprint
    rh_oidc_provider_url = ocm_cluster_rosa_classic.rosa_sts_cluster.sts.oidc_endpoint_url
    operator_role_properties = each.value
}


resource "aws_iam_role" "operator_role" {
  count = length(data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles)

  name = data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles[count.index].role_name
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Condition = {
            StringEquals = {
                "${ocm_cluster_rosa_classic.rosa_sts_cluster.sts.oidc_endpoint_url}:sub" = data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles[count.index].service_accounts
            }
        }
        Principal = {
          Federated = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${ocm_cluster_rosa_classic.rosa_sts_cluster.sts.oidc_endpoint_url}"
        }
      },
    ]
  })

  tags = {
    red-hat-managed = true
    rosa_cluster_id = ocm_cluster_rosa_classic.rosa_sts_cluster.id
    operator_namespace = data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles[count.index].namespace
    operator_name = data.ocm_rosa_operator_roles.operator_roles.operator_iam_roles[count.index].operator_name
  }
}


