resource "aws_iam_role" "cloud_network_config_role" {
  count = length(var.clusters)
  name = "${var.clusters[count.index].operator_role_prefix}-openshift-cloud-network-config-controller-c"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Condition = {
            StringEquals = {
                "${var.rh_oidc_provider_url}:sub" = ["system:serviceaccount:openshift-cloud-network-config-controller:cloud-network-config-controller"]
            }
        }
        Principal = {
          Federated = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/${var.rh_oidc_provider_url}"
        }
      },
    ]
  })

  tags = {
    red-hat-managed = true
    rosa_cluster_id = var.clusters[count.index].id
    operator_namespace = "openshift-cloud-network-config-controller"
    operator_name = "cloud-credentials"
  }
}

resource "aws_iam_role_policy_attachment" "cloud_network_config_role_policy_attachment" {
  count = length(var.clusters)
  role = aws_iam_role.cloud_network_config_role[count.index].name
  policy_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/ManagedOpenShift-openshift-cloud-network-config-controller-cloud"
}