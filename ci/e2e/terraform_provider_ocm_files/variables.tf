variable token {
  type = string
  sensitive = true
}

variable url {
    type = string
    default = "https://api.stage.openshift.com"
}

variable operator_role_prefix {
    type = string
}

variable account_role_prefix {
    type = string
}

variable cluster_name {
    type = string
}
