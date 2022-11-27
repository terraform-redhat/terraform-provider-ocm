## 0.1.7 (Jun 07, 2022)

ENHANCEMENTS:

* Add `ocm_cluster_rosa_classic` resource to create ROSA STS clusters.

## 0.1.6 (Jun 07, 2022)

ENHANCEMENTS:

* Build for Apple M1.


## 0.1.5 (December 01, 2021)

ENHANCEMENTS:

* Add `search` and `order` to cloud providers data source.

* Add `search` and `order` to versions data source.


## 0.1.4 (November 24, 2021)

FEATURES:

* Add `ocm_machine_pool` resource.

* Add `ocm_groups` data source.

* Add `ocm_group_membership` resource.

* Add `ocm_versions` data source.

ENHANCEMENTS:

* Add `api_url` and `console_url` read-only attributes.

* Add support for specifying the number and type of compute nodes with the
  `compute_nodes` and `compute_machine_type` attributes.

* Add support for selection the _OpenShift_ version with the `version`
  attribute.

* Add support for configuring cluster networks with attributes `machine_cidr`,
  `service_cidr`, `pod_cidr` and `host_prefix`.

* Add support for _CCS_ clusters with the `ccs_enabled` attribute.

BREAKING CHANGES:

* Renamed attribute `cluster_id` to `cluster`.
