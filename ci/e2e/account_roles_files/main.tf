terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.20.0"
    }
  }
}

# create account roles
module account_role{
  source = "terraform-redhat/rosa-sts/aws"
  version = "0.0.3"

  create_operator_roles = false
  create_oidc_provider = false
  create_account_roles = true

  account_role_prefix = var.account_role_prefix
  ocm_environment = var.ocm_env
  rosa_openshift_version=var.rosa_openshift_version
}

