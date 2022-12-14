/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/terraform-provider-ocm/build"
	"github.com/openshift-online/terraform-provider-ocm/provider/cloudprovider"
	"github.com/openshift-online/terraform-provider-ocm/provider/clusterresource"
	"github.com/openshift-online/terraform-provider-ocm/provider/util"
	"os"
)

// Config contains the configuration of the provider.
type Config struct {
	URL              string
	TokenURL         string
	User             string
	Password         string
	Token            string
	ClientID         string
	ClientSecret     string
	TrustedCAs       string
	TerraformVersion string
	Insecure         bool
}

// Provider creates the schema for the provider.
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Description: "URL of the API server.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"token_url": {
				Description: "OpenID token URL.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"user": {
				Description: "User name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"password": {
				Description: "User password.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"token": {
				Description: "Access or refresh token.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"client_id": {
				Description: "OpenID client identifier.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"client_secret": {
				Description: "OpenID client secret.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"trusted_cas": {
				Description: "PEM encoded certificates of authorities that will " +
					"be trusted. If this isn't explicitly specified then " +
					"the provider will trust the certificate authorities " +
					"trusted by default by the system.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"insecure": {
				Description: "When set to 'true' enables insecure communication " +
					"with the server. This disables verification of TLS " +
					"certificates and host names and it isn't recommended " +
					"for production environments.",
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"ocm_cluster":              clusterresource.ResourceCluster,
			"ocm_cluster_rosa_classic": &ClusterRosaClassicResourceType{},
			"ocm_group_membership":     &GroupMembershipResourceType{},
			"ocm_identity_provider":    &IdentityProviderResourceType{},
			"ocm_machine_pool":         &MachinePoolResourceType{},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ocm_cloud_providers":     cloudprovider.DataSourceCloudProvider(),
			"ocm_rosa_operator_roles": &RosaOperatorRolesDataSourceType{},
			"ocm_groups":              &GroupsDataSourceType{},
			"ocm_machine_types":       &MachineTypesDataSourceType{},
			"ocm_versions":            &VersionsDataSourceType{},
		},
	}
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, p, d)
	}

	return p
}

func providerConfigure(ctx context.Context, provider *schema.Provider, d *schema.ResourceData) (*sdk.Connection, diag.Diagnostics) {
	terraformVersion := provider.TerraformVersion
	if terraformVersion == "" {
		// Terraform 0.12 introduced this field to the protocol
		// We can therefore assume that if it's missing it's 0.10 or 0.11
		terraformVersion = "0.11+compatible"
	}
	config := Config{
		TerraformVersion: terraformVersion,
	}

	if v, ok := d.GetOk("url"); ok {
		config.URL = v.(string)
	}

	if v, ok := d.GetOk("token_url"); ok {
		config.TokenURL = v.(string)
	}

	if v, ok := d.GetOk("user"); ok {
		config.User = v.(string)
	}

	if v, ok := d.GetOk("password"); ok {
		config.Password = v.(string)
	}

	if v, ok := d.GetOk("token"); ok {
		config.Token = v.(string)
	}

	if v, ok := d.GetOk("client_id"); ok {
		config.ClientID = v.(string)
	}

	if v, ok := d.GetOk("client_secret"); ok {
		config.ClientSecret = v.(string)
	}

	if v, ok := d.GetOk("trusted_cas"); ok {
		config.TrustedCAs = v.(string)
	}

	if v, ok := d.GetOk("insecure"); ok {
		config.Insecure = v.(bool)
	}

	// Create the builder:
	builder := sdk.NewConnectionBuilder()
	builder.Logger(util.Logger)
	builder.Agent(fmt.Sprintf("OCM-TF/%s-%s", build.Version, build.Commit))

	// Copy the settings:
	if config.URL != "" {
		builder.URL(config.URL)
	} else {
		url, ok := os.LookupEnv("OCM_URL")
		if ok {
			builder.URL(url)
		}
	}
	if config.TokenURL != "" {
		builder.TokenURL(config.TokenURL)
	}
	if config.User != "" && config.Password != "" {
		builder.User(config.User, config.Password)
	}
	if config.Token != "" {
		builder.Tokens(config.Token)
	} else {
		token, ok := os.LookupEnv("OCM_TOKEN")
		if ok {
			builder.Tokens(token)
		}
	}
	if config.ClientID != "" && config.ClientSecret != "" {
		builder.Client(config.ClientID, config.ClientSecret)
	}
	if config.Insecure {
		builder.Insecure(config.Insecure)
	}
	if config.TrustedCAs != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(config.TrustedCAs)) {
			return nil, diag.Errorf("the value of 'trusted_cas' doesn't contain any certificate")
		}
		builder.TrustedCAs(pool)
	}

	// Create the connection:
	connection, err := builder.BuildContext(ctx)
	if err != nil {
		return nil, diag.Errorf(err.Error())
	}

	return connection, nil
}
