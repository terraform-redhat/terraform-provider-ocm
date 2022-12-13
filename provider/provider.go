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
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"

	"github.com/openshift-online/terraform-provider-ocm/build"
)

// Provider is the implementation of the Provider.
type Provider struct {
	logger     logging.Logger
	connection *sdk.Connection
}

// Config contains the configuration of the provider.
type Config struct {
	URL          types.String `tfsdk:"url"`
	TokenURL     types.String `tfsdk:"token_url"`
	User         types.String `tfsdk:"user"`
	Password     types.String `tfsdk:"password"`
	Token        types.String `tfsdk:"token"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	TrustedCAs   types.String `tfsdk:"trusted_cas"`
	Insecure     types.Bool   `tfsdk:"insecure"`
}

// New creates the provider.
func New() provider.Provider {
	return &Provider{}
}

// Provider creates the schema for the provider.
func (p *Provider) GetSchema(ctx context.Context) (schema tfsdk.Schema, diags diag.Diagnostics) {
	schema = tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"url": {
				Description: "URL of the API server.",
				Type:        types.StringType,
				Optional:    true,
			},
			"token_url": {
				Description: "OpenID token URL.",
				Type:        types.StringType,
				Optional:    true,
			},
			"user": {
				Description: "User name.",
				Type:        types.StringType,
				Optional:    true,
			},
			"password": {
				Description: "User password.",
				Type:        types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"token": {
				Description: "Access or refresh token.",
				Type:        types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"client_id": {
				Description: "OpenID client identifier.",
				Type:        types.StringType,
				Optional:    true,
			},
			"client_secret": {
				Description: "OpenID client secret.",
				Type:        types.StringType,
				Optional:    true,
				Sensitive:   true,
			},
			"trusted_cas": {
				Description: "PEM encoded certificates of authorities that will " +
					"be trusted. If this isn't explicitly specified then " +
					"the provider will trust the certificate authorities " +
					"trusted by default by the system.",
				Type:     types.StringType,
				Optional: true,
			},
			"insecure": {
				Description: "When set to 'true' enables insecure communication " +
					"with the server. This disables verification of TLS " +
					"certificates and host names and it isn't recommended " +
					"for production environments.",
				Type:     types.BoolType,
				Optional: true,
			},
		},
	}
	return
}

// configure is the configuration function of the provider. It is responsible for checking the
// connection parameters and creating the connection that will be used by the resources.
func (p *Provider) Configure(ctx context.Context, request provider.ConfigureRequest,
	response *provider.ConfigureResponse) {
	// Retrieve the provider configuration:
	var config Config
	diags := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Determine the log level used by the SDK from the environment variables used by Terraform:
	level := os.Getenv("TF_LOG")

	// The plugin infrastructure redirects the log package output so that it is sent to the main
	// Terraform process, so if we want to have the logs of the SDK redirected we need to use
	// the log package as well.
	logger, err := logging.NewGoLoggerBuilder().
		Error(true).
		Warn(true).
		Info(true).
		Debug(strings.EqualFold(level, "DEBUG")).
		Build()
	if err != nil {
		response.Diagnostics.AddError(err.Error(), "")
		return
	}

	// Create the builder:
	builder := sdk.NewConnectionBuilder()
	builder.Logger(logger)
	builder.Agent(fmt.Sprintf("OCM-TF/%s-%s", build.Version, build.Commit))

	// Copy the settings:
	if !config.URL.Null {
		builder.URL(config.URL.Value)
	} else {
		url, ok := os.LookupEnv("OCM_URL")
		if ok {
			builder.URL(url)
		}
	}
	if !config.TokenURL.Null {
		builder.TokenURL(config.TokenURL.Value)
	}
	if !config.User.Null && !config.Password.Null {
		builder.User(config.User.Value, config.Password.Value)
	}
	if !config.Token.Null {
		builder.Tokens(config.Token.Value)
	} else {
		token, ok := os.LookupEnv("OCM_TOKEN")
		if ok {
			builder.Tokens(token)
		}
	}
	if !config.ClientID.Null && !config.ClientSecret.Null {
		builder.Client(config.ClientID.Value, config.ClientSecret.Value)
	}
	if !config.Insecure.Null {
		builder.Insecure(config.Insecure.Value)
	}
	if !config.TrustedCAs.Null {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(config.TrustedCAs.Value)) {
			response.Diagnostics.AddError(
				"the value of 'trusted_cas' doesn't contain any certificate",
				"",
			)
			return
		}
		builder.TrustedCAs(pool)
	}

	// Create the connection:
	connection, err := builder.BuildContext(ctx)
	if err != nil {
		response.Diagnostics.AddError(err.Error(), "")
		return
	}

	// Save the connection:
	p.logger = logger
	p.connection = connection
}

// GetResources returns the resources supported by the provider.
func (p *Provider) GetResources(ctx context.Context) (result map[string]provider.ResourceType,
	diags diag.Diagnostics) {
	result = map[string]provider.ResourceType{
		"ocm_cluster":              &ClusterResourceType{},
		"ocm_cluster_rosa_classic": &ClusterRosaClassicResourceType{},
		"ocm_group_membership":     &GroupMembershipResourceType{},
		"ocm_identity_provider":    &IdentityProviderResourceType{},
		"ocm_machine_pool":         &MachinePoolResourceType{},
	}
	return
}

// GetDataSources returns the data sources supported by the provider.
func (p *Provider) GetDataSources(ctx context.Context) (result map[string]provider.DataSourceType,
	diags diag.Diagnostics) {
	result = map[string]provider.DataSourceType{
		"ocm_cloud_providers":     &CloudProvidersDataSourceType{},
		"ocm_rosa_operator_roles": &RosaOperatorRolesDataSourceType{},
		"ocm_groups":              &GroupsDataSourceType{},
		"ocm_machine_types":       &MachineTypesDataSourceType{},
		"ocm_versions":            &VersionsDataSourceType{},
	}
	return
}
