output ocm_role_arns {
    description = "ocm roles' arns"
    value = module.ocm_role.ocm_role_arns
}

output ocm_user_role_arns {
    description = "ocm user roles' arns"
    value = module.user_role.ocm_user_role_arns
}