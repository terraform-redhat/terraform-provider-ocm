# Account IAM roles creation example

As a prerequisite for ROSA STS clusters, 
this example demonstrates the process of creating account IAM roles.
To run it:

* Decide STS account_role_prefix    
    ```
    export TF_VAR_account_role_prefix=...
    ```

* Provide STS openshift_version
    ```
    export TF_VAR_openshift_versionx=...
    
* Provide STS ocm_environment
    ```
    export TF_VAR_ocm_environment=...
    ```
and then run the `terraform apply` command.

