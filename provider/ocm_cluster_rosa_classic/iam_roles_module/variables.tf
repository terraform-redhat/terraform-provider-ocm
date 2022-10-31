variable redhat_aws_account_id {
    type = string
    default = "710019948333"
}

variable rosa_openshift_version {
    type = string
    default = "4.10"
}

variable create_account_roles {
    type = bool
    default = false
}

variable ocm_orgs {
    description = "ocm orgs"
    type = list(object({
        org_id = string
        external_id = string
    }))
    default = []
}

variable ocm_users {
    description = "ocm users account id and username"
    type = list(object({
        id = string
        user_name = string
    }))
    default = []
}

variable clusters {
    description = "clusters information for operator roles"
    type = list(object({
        id = string
        operator_role_prefix = string
    }))
    default = []
}

variable rh_oidc_provider_url {
    description = "oidc provider url"
    type = string
    default = "rh-oidc.s3.us-east-1.amazonaws.com"
}

variable rh_oidc_provider_thumbprint {
    description = "Thumbprint for https://rh-oidc.s3.us-east-1.amazonaws.com"
    type = string
    default = "917e732d330f9a12404f73d8bea36948b929dffc"
}
