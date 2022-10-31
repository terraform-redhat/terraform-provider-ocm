# terraform-aws-rosa-sts-roles

Create rosa roles, policies and identity provider in an declarative way
Terraform AWS ROSA STS Roles

In order to deploy [ROSA](https://docs.openshift.com/rosa/welcome/index.html) with [STS](https://docs.openshift.com/rosa/rosa_planning/rosa-sts-aws-prereqs.html), AWS Account needs to have the following roles placed:

* Account Roles (One per AWS account)
* OCM Roles (For OCM UI, One per OCM Org)
* User Role (For OCM UI, One per user live in ocm org)
* Operator Roles (One Per Cluster)
* OIDC Identity Provider (One Per Cluster)

This terraform module tries to replicate rosa CLI roles/policies creation so that:

* Users have a declartive way to create AWS roles & Policies.
* Users can implement security/infrastructure as code practices.
* Batch creation of user roles, ocm roles and operator roles.

## Prerequisites

* AWS Admin Account
* OCM Account and OCM CLI
* ROSA CLI

## Get OCM Information

When create OCM roles, the roles require org id and external id.

When create User role. the roles require user id and username.

The information can be retrieved from ocm cli.
```
ocm whoami
{
  "kind": "Account",
  "id": "26kcPSEHi0Y6MkTS7OowxfFYmZo",
  "href": "/api/accounts_mgmt/v1/accounts/26kcPSEHi0Y6MkTS7OowxfFYmZo",
  "created_at": "2022-03-22T17:43:19Z",
  "email": "shading+mobb@redhat.com",
  "first_name": "Shaozhen",
  "last_name": "Ding",
  "organization": {
    "kind": "Organization",
    "id": "1rkxPO7W12geIcRWITwI0I8VIQV",
    "href": "/api/accounts_mgmt/v1/organizations/1rkxPO7W12geIcRWITwI0I8VIQV",
    "created_at": "2021-04-27T14:31:03Z",
    "ebs_account_id": "7113273",
    "external_id": "14540493",
    "name": "Red Hat, Inc.",
    "updated_at": "2022-06-16T14:17:02Z"
  },
  "updated_at": "2022-05-25T02:02:02Z",
  "username": "shading_mobb"
}
```

## Get Clusters Information.

In order to create operator roles for clusters. Users need to provide cluster id, operator role prefix and OIDC Endpoint URL

```
 rosa describe cluster -c shaozhenprivate -o json
{
  "kind": "Cluster",
  "id": "1srtno3qggal8ujsegvtb2njvbmhdu8c",
  "href": "/api/clusters_mgmt/v1/clusters/1srtno3qggal8ujsegvtb2njvbmhdu8c",
  "aws": {
    "sts": {
      "oidc_endpoint_url": "https://rh-oidc.s3.us-east-1.amazonaws.com/1srtno3qggal8ujsegvtb2njvbmhdu8c",
      "operator_iam_roles": [
        {
          "id": "",
          "name": "ebs-cloud-credentials",
          "namespace": "openshift-cluster-csi-drivers",
          "role_arn": "arn:aws:iam::${AWS_ACCOUNT_ID}:role/shaozhenprivate-w4e1-openshift-cluster-csi-drivers-ebs-cloud-cre",
          "service_account": ""
        },
```

In the above example:

* cluster_id =  1srtno3qggal8ujsegvtb2njvbmhdu8c
* operator_role_prefix = shaozhenprivate-w4e1
* rh_oidc_endpoint_url = rh-oidc.s3.us-east-1.amazonaws.com

## Usage

### Sample Usage

```
module sts_roles {
    source = "rh-mobb/rosa-sts-roles/aws"
    ## Whether to Create Account Roles & Policies
    create_account_roles = false
    ## Create a list of ocm roles. The org id and external id retrieve from OCM
    # ocm_orgs = [{
    #     org_id = "1rkxPO7W12geIcRWITwI0I8VIQV"
    #     external_id = "14540493"
    # }]
    ## Create a list of user roles. The user id and user name can retrieve from OCM
    ocm_users = [{
        id = "26kcPSEHi0Y6MkTS7OowxfFYmZo"
        user_name = "shading_mobb"
    }]
    ## Create a list of operator roles for clusters. The cluster id and operator_role_prefix can retrieve from OCM.
    clusters = [{
        id = "1ssjjr1b2npkg9c70e8kqehfeqmscqeu"
        operator_role_prefix = "shaozhenprivate-w4e1"
    }]
    rh_oidc_provider_url = "rh-oidc.s3.us-east-1.amazonaws.com/1srtno3qggal8ujsegvtb2njvbmhdu8c" 
    rh_oidc_provider_thumbprint = "917e732d330f9a12404f73d8bea36948b929dffc"
}
```
