# ROSA STS cluster creation example

This example shows how to create an STS _ROSA_ cluster. _ROSA_ stands for Red Hat Openshift Service on AWS
and is a cluster that is created in the AWS cloud infrastructure.

To run it:

* Provide OCM Authentication Token 

  OCM authentication token that you can get [here](https://console.redhat.com/openshift/token).
    ```
    export TF_VAR_token=...
    ```

* Provide OCM environment by setting a value to url    
    ```
    export TF_VAR_url=...
    ```

* Decide STS operator_role_prefix    
    ```
    export TF_VAR_operator_role_prefix=...
    ```

* Provide OIDC endpoint url
    ```
    export TF_VAR_oidc_endpoint_url=...
    ```

* Provide OIDC Private Key Secret ARN
    ```
    export TF_VAR_oidc_private_key_secret_arn=...
    ```

and then run the `terraform apply` command.

