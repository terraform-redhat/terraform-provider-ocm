terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

module account_role {
    count = var.create_account_roles ? 1 : 0
    source = "./account_roles"
    redhat_aws_account_id = var.redhat_aws_account_id
    rosa_openshift_version = var.rosa_openshift_version
}

module ocm_role {
    source = "./ocm_role"
    redhat_aws_account_id = var.redhat_aws_account_id
    ocm_orgs = var.ocm_orgs
}

module user_role {
    source = "./user_role"
    redhat_aws_account_id = var.redhat_aws_account_id
    ocm_users = var.ocm_users
}

module operator_role {
    source = "./operator_roles"
    clusters = var.clusters
    rh_oidc_provider_thumbprint = var.rh_oidc_provider_thumbprint
    rh_oidc_provider_url = var.rh_oidc_provider_url
}
