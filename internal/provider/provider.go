package provider

import (
	"context"
	"fmt"
	"math"
	"net/url"

	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &RestAPIProvider{}
var _ provider.ProviderWithFunctions = &RestAPIProvider{}
var _ provider.ProviderWithEphemeralResources = &RestAPIProvider{}

// RestAPIProvider defines the provider implementation
type RestAPIProvider struct {
	version string
}

type RestAPIProviderModel struct {
	URI                 types.String          `tfsdk:"uri"`
	Insecure            types.Bool            `tfsdk:"insecure"`
	Username            types.String          `tfsdk:"username"`
	Password            types.String          `tfsdk:"password"`
	BearerToken         types.String          `tfsdk:"bearer_token"`
	Headers             types.Map             `tfsdk:"headers"`
	UseCookies          types.Bool            `tfsdk:"use_cookies"`
	Timeout             types.Int64           `tfsdk:"timeout"`
	IDAttribute         types.String          `tfsdk:"id_attribute"`
	CreateMethod        types.String          `tfsdk:"create_method"`
	ReadMethod          types.String          `tfsdk:"read_method"`
	UpdateMethod        types.String          `tfsdk:"update_method"`
	DestroyMethod       types.String          `tfsdk:"destroy_method"`
	CopyKeys            types.List            `tfsdk:"copy_keys"`
	WriteReturnsObject  types.Bool            `tfsdk:"write_returns_object"`
	CreateReturnsObject types.Bool            `tfsdk:"create_returns_object"`
	XSSIPrefix          types.String          `tfsdk:"xssi_prefix"`
	RateLimit           types.Float64         `tfsdk:"rate_limit"`
	TestPath            types.String          `tfsdk:"test_path"`
	Debug               types.Bool            `tfsdk:"debug"`
	CertString          types.String          `tfsdk:"cert_string"`
	KeyString           types.String          `tfsdk:"key_string"`
	CertFile            types.String          `tfsdk:"cert_file"`
	KeyFile             types.String          `tfsdk:"key_file"`
	RootCAFile          types.String          `tfsdk:"root_ca_file"`
	RootCAString        types.String          `tfsdk:"root_ca_string"`
	OAuthClientCreds    *OAuthClientDataModel `tfsdk:"oauth_client_credentials"`
	RetriesConfig       *RetriesDataModel     `tfsdk:"retries"`
}

type OAuthClientDataModel struct {
	OAuthClientID      types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret  types.String `tfsdk:"oauth_client_secret"`
	OAuthTokenEndpoint types.String `tfsdk:"oauth_token_endpoint"`
	OAuthScopes        types.List   `tfsdk:"oauth_scopes"`
	EndpointParams     types.Map    `tfsdk:"endpoint_params"`
}

type RetriesDataModel struct {
	MaxRetries types.Int64 `tfsdk:"max_retries"`
	MinWait    types.Int64 `tfsdk:"min_wait"`
	MaxWait    types.Int64 `tfsdk:"max_wait"`
}

type ProviderData struct {
	client *apiclient.APIClient
	opts   *apiclient.APIClientOpt
}

// GetClient returns the API client, creating it if necessary.
// This allows for lazy initialization when lazy parameters become available.
func (pd *ProviderData) GetClient() (*apiclient.APIClient, error) {
	if pd.client != nil {
		return pd.client, nil
	}

	if pd.opts == nil {
		return nil, fmt.Errorf("provider configuration not available")
	}

	if pd.opts.URI == "" {
		return nil, fmt.Errorf("provider URI is not set - it may depend on a resource that hasn't been created yet")
	}

	client, err := apiclient.NewAPIClient(pd.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	pd.client = client
	return client, nil
}

func (p *RestAPIProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "restapi"
	resp.Version = p.version
}

// Provider implements the REST API provider
func (p *RestAPIProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for interacting with RESTful APIs.",
		Attributes: map[string]schema.Attribute{
			"uri": schema.StringAttribute{
				Optional:    true,
				Description: "URI of the REST API endpoint. This serves as the base of all requests.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "When using https, this disables TLS verification of the host.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "When set, will use this username for BASIC auth to the API.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "When set, will use this password for BASIC auth to the API.",
			},
			"bearer_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Token to use for Authorization: Bearer <token>",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "A map of header names and values to set on all outbound requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token or change Content-Type from `application/json`. If `username` and `password` are set and Authorization is one of the headers defined here, the BASIC auth credentials are discarded.",
			},
			"use_cookies": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable cookie jar to persist session.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "When set, will cause requests taking longer than this time (in seconds) to be aborted. Must be a positive integer.",
			},
			"id_attribute": schema.StringAttribute{
				Optional:    true,
				Description: "When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to http://foo.com/bar/VALUE_OF_NAME. This value may also be a '/'-delimeted path to the id attribute if it is multple levels deep in the data (such as `attributes/id` in the case of an object `{ \"attributes\": { \"id\": 1234 }, \"config\": { \"name\": \"foo\", \"something\": \"bar\"}}`",
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `POST`. The HTTP method used to CREATE objects of this type on the API server.",
				Optional:    true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `GET`. The HTTP method used to READ objects of this type on the API server.",
				Optional:    true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `PUT`. The HTTP method used to UPDATE objects of this type on the API server.",
				Optional:    true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `DELETE`. The HTTP method used to DELETE objects of this type on the API server.",
				Optional:    true,
			},
			"copy_keys": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "When set, any PUT to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.",
			},
			"write_returns_object": schema.BoolAttribute{
				Optional:    true,
				Description: "Set this when the API returns the object created on all write operations (POST, PUT). This is used by the provider to refresh internal data structures.",
			},
			"create_returns_object": schema.BoolAttribute{
				Optional:    true,
				Description: "Set this when the API returns the object created only on creation operations (POST). This is used by the provider to refresh internal data structures.",
			},
			"xssi_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Trim the xssi prefix from response string, if present, before parsing.",
			},
			"rate_limit": schema.Float64Attribute{
				Optional:    true,
				Description: "Set this to limit the number of requests per second made to the API. Must be a positive number.",
			},
			"test_path": schema.StringAttribute{
				Optional:    true,
				Description: "If set, the provider will issue a read_method request to this path after instantiation requiring a 200 OK response before proceeding. This is useful if your API provides a no-op endpoint that can signal if this provider is configured correctly. Response data will be ignored.",
			},
			"debug": schema.BoolAttribute{
				Optional:    true,
				Description: "Enabling this will cause the HTTP request and response to be printed to STDOUT by the API client regardless of the Terraform TFLOG settings.",
			},
			"cert_string": schema.StringAttribute{
				Optional:    true,
				Description: "When set with the key_string parameter, the provider will load a client certificate as a string for mTLS authentication.",
			},
			"key_string": schema.StringAttribute{
				Optional:    true,
				Description: "When set with the cert_string parameter, the provider will load a client certificate as a string for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
			},
			"root_ca_string": schema.StringAttribute{
				Optional:    true,
				Description: "When set, the provider will load a root CA certificate as a string for mTLS authentication. This is useful when the API server is using a self-signed certificate and the client needs to trust it.",
			},
			"cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "When set with the key_file parameter, the provider will load a client certificate as a file for mTLS authentication.",
			},
			"key_file": schema.StringAttribute{
				Optional:    true,
				Description: "When set with the cert_file parameter, the provider will load a client certificate as a file for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
			},
			"root_ca_file": schema.StringAttribute{
				Optional:    true,
				Description: "When set, the provider will load a root CA certificate as a file for mTLS authentication. This is useful when the API server is using a self-signed certificate and the client needs to trust it.",
			},
		},
		Blocks: map[string]schema.Block{
			"oauth_client_credentials": schema.SingleNestedBlock{
				Description: "Configuration for oauth client credential flow using the https://pkg.go.dev/golang.org/x/oauth2 implementation",
				Attributes: map[string]schema.Attribute{
					"oauth_client_id": schema.StringAttribute{
						Description: "client id",
						Optional:    true,
					},
					"oauth_client_secret": schema.StringAttribute{
						Description: "client secret",
						Optional:    true,
					},
					"oauth_token_endpoint": schema.StringAttribute{
						Description: "oauth token endpoint",
						Optional:    true,
					},
					"oauth_scopes": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "scopes",
					},
					"endpoint_params": schema.MapAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Additional key/values to pass to the underlying Oauth client library (as EndpointParams)",
					},
				},
			},
			"retries": schema.SingleNestedBlock{
				Description: "Configuration for automatic retry (connection/TLS/etc errors or a 500-range response except 501) of failed HTTP requests",
				Attributes: map[string]schema.Attribute{
					"max_retries": schema.Int64Attribute{
						Description: "Maximum number of retries for failed requests. Defaults to 0.",
						Optional:    true,
					},
					"min_wait": schema.Int64Attribute{
						Description: "Minimum wait time in seconds between retries. Defaults to 1.",
						Optional:    true,
					},
					"max_wait": schema.Int64Attribute{
						Description: "Maximum wait time in seconds between retries. Defaults to 30.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (p *RestAPIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RestAPIProviderModel

	// Populate the data model, and add it to Diagnostics
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract headers from the map
	headers := make(map[string]string)
	if !data.Headers.IsNull() && !data.Headers.IsUnknown() {
		diags := data.Headers.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Extract copy_keys from the list
	var copyKeys []string
	if !data.CopyKeys.IsNull() && !data.CopyKeys.IsUnknown() {
		diags := data.CopyKeys.ElementsAs(ctx, &copyKeys, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Check for conflicting authentication methods
	username := existingOrEnvOrDefaultString(&resp.Diagnostics, "username", data.Username, "REST_API_USERNAME", "", false)
	password := existingOrEnvOrDefaultString(&resp.Diagnostics, "password", data.Password, "REST_API_PASSWORD", "", false)
	bearerToken := existingOrEnvOrDefaultString(&resp.Diagnostics, "bearer_token", data.BearerToken, "REST_API_BEARER", "", false)

	if username != "" && password != "" && bearerToken != "" {
		resp.Diagnostics.AddError(
			"Conflicting Authentication Methods",
			"Both basic auth (username/password) and bearer_token are set - please set only one authentication method",
		)
		return
	}

	// Add bearer token to headers if provided
	if bearerToken != "" {
		headers["Authorization"] = "Bearer " + bearerToken
	}

	// Populate default options
	opt := &apiclient.APIClientOpt{
		URI:                 existingOrEnvOrDefaultString(&resp.Diagnostics, "uri", data.URI, "REST_API_URI", "", true),
		Insecure:            existingOrEnvOrDefaultBool(&resp.Diagnostics, "insecure", data.Insecure, "REST_API_INSECURE", false, false),
		Username:            username,
		Password:            password,
		Headers:             headers,
		UseCookies:          existingOrEnvOrDefaultBool(&resp.Diagnostics, "use_cookies", data.UseCookies, "REST_API_USE_COOKIES", false, false),
		Timeout:             existingOrEnvOrDefaultInt(&resp.Diagnostics, "timeout", data.Timeout, "REST_API_TIMEOUT", 60, false),
		IDAttribute:         existingOrEnvOrDefaultString(&resp.Diagnostics, "id_attribute", data.IDAttribute, "REST_API_ID_ATTRIBUTE", "id", false),
		CopyKeys:            copyKeys,
		WriteReturnsObject:  existingOrEnvOrDefaultBool(&resp.Diagnostics, "write_returns_object", data.WriteReturnsObject, "REST_API_WRO", false, false),
		CreateReturnsObject: existingOrEnvOrDefaultBool(&resp.Diagnostics, "create_returns_object", data.CreateReturnsObject, "REST_API_CRO", false, false),
		XSSIPrefix:          existingOrEnvOrDefaultString(&resp.Diagnostics, "xssi_prefix", data.XSSIPrefix, "REST_API_XSSI_PREFIX", "", false),
		RateLimit:           existingOrEnvOrDefaultFloat(&resp.Diagnostics, "rate_limit", data.RateLimit, "REST_API_RATE_LIMIT", math.MaxFloat64, false),
		Debug:               existingOrEnvOrDefaultBool(&resp.Diagnostics, "debug", data.Debug, "REST_API_DEBUG", false, false),
		CreateMethod:        existingOrEnvOrDefaultString(&resp.Diagnostics, "create_method", data.CreateMethod, "REST_API_CREATE_METHOD", "POST", false),
		ReadMethod:          existingOrEnvOrDefaultString(&resp.Diagnostics, "read_method", data.ReadMethod, "REST_API_READ_METHOD", "GET", false),
		UpdateMethod:        existingOrEnvOrDefaultString(&resp.Diagnostics, "update_method", data.UpdateMethod, "REST_API_UPDATE_METHOD", "PUT", false),
		DestroyMethod:       existingOrEnvOrDefaultString(&resp.Diagnostics, "destroy_method", data.DestroyMethod, "REST_API_DESTROY_METHOD", "DELETE", false),
		CertFile:            existingOrEnvOrDefaultString(&resp.Diagnostics, "cert_file", data.CertFile, "REST_API_CERT_FILE", "", false),
		KeyFile:             existingOrEnvOrDefaultString(&resp.Diagnostics, "key_file", data.KeyFile, "REST_API_KEY_FILE", "", false),
		CertString:          existingOrEnvOrDefaultString(&resp.Diagnostics, "cert_string", data.CertString, "REST_API_CERT_STRING", "", false),
		KeyString:           existingOrEnvOrDefaultString(&resp.Diagnostics, "key_string", data.KeyString, "REST_API_KEY_STRING", "", false),
		RootCAFile:          existingOrEnvOrDefaultString(&resp.Diagnostics, "root_ca_file", data.RootCAFile, "REST_API_ROOT_CA_FILE", "", false),
		RootCAString:        existingOrEnvOrDefaultString(&resp.Diagnostics, "root_ca_string", data.RootCAString, "REST_API_ROOT_CA_STRING", "", false),
	}

	// Handle retries configuration
	if data.RetriesConfig != nil {
		opt.RetryMax = existingOrEnvOrDefaultInt(&resp.Diagnostics, "retries.max_retries", data.RetriesConfig.MaxRetries, "REST_API_RETRY_MAX", 0, false)
		opt.RetryWaitMin = existingOrEnvOrDefaultInt(&resp.Diagnostics, "retries.min_wait", data.RetriesConfig.MinWait, "REST_API_RETRY_WAIT_MIN", 0, false)
		opt.RetryWaitMax = existingOrEnvOrDefaultInt(&resp.Diagnostics, "retries.max_wait", data.RetriesConfig.MaxWait, "REST_API_RETRY_WAIT_MAX", 0, false)
	}

	if _, err := url.Parse(opt.URI); err != nil {
		resp.Diagnostics.AddError(
			"Invalid URI Configuration",
			fmt.Sprintf("The uri configuration value must be a valid URI. The value '%s' is not valid: %s", opt.URI, err.Error()),
		)
	}

	if opt.Timeout < 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			fmt.Sprintf("The timeout configuration value must be a positive integer. The value %d is not valid.", opt.Timeout),
		)
	}

	if opt.RateLimit <= 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			fmt.Sprintf("The rate_limit configuration value must be a positive number. The value %f is not valid.", opt.RateLimit),
		)
	}

	if opt.RetryMax < 0 {
		resp.Diagnostics.AddError(
			"Invalid Retry Configuration",
			fmt.Sprintf("The retries.max_retries value must be non-negative. The value %d is not valid.", opt.RetryMax),
		)
	}
	if opt.RetryWaitMin < 0 {
		resp.Diagnostics.AddError(
			"Invalid Retry Configuration",
			fmt.Sprintf("The retries.min_wait value must be non-negative. The value %d is not valid.", opt.RetryWaitMin),
		)
	}
	if opt.RetryWaitMax < 0 {
		resp.Diagnostics.AddError(
			"Invalid Retry Configuration",
			fmt.Sprintf("The retries.max_wait value must be non-negative. The value %d is not valid.", opt.RetryWaitMax),
		)
	}
	if opt.RetryWaitMin > 0 && opt.RetryWaitMax > 0 && opt.RetryWaitMin > opt.RetryWaitMax {
		resp.Diagnostics.AddError(
			"Invalid Retry Configuration",
			fmt.Sprintf("The retries.min_wait (%d) must be less than or equal to retries.max_wait (%d).", opt.RetryWaitMin, opt.RetryWaitMax),
		)
	}

	// Check for conflicting OAuth and basic auth
	if opt.OAuthClientID != "" && opt.Username != "" {
		resp.Diagnostics.AddError(
			"Conflicting Authentication Methods",
			"Both OAuth credentials and basic auth (username/password) are configured. Please use only one authentication method.",
		)
		return
	}

	// Check for conflicting certificate configurations
	if opt.CertFile != "" && opt.CertString != "" {
		resp.Diagnostics.AddError(
			"Conflicting Certificate Configuration",
			"Both cert_file and cert_string are set. Please use only one method to provide the certificate.",
		)
		return
	}
	if opt.KeyFile != "" && opt.KeyString != "" {
		resp.Diagnostics.AddError(
			"Conflicting Key Configuration",
			"Both key_file and key_string are set. Please use only one method to provide the key.",
		)
		return
	}
	if opt.RootCAFile != "" && opt.RootCAString != "" {
		resp.Diagnostics.AddError(
			"Conflicting Root CA Configuration",
			"Both root_ca_file and root_ca_string are set. Please use only one method to provide the root CA.",
		)
		return
	}

	// Validate cert/key pairs
	if (opt.CertFile != "" || opt.CertString != "") && (opt.KeyFile == "" && opt.KeyString == "") {
		resp.Diagnostics.AddError(
			"Incomplete Certificate Configuration",
			"A certificate is provided but no key is configured. Both cert and key must be provided for mTLS.",
		)
		return
	}
	if (opt.KeyFile != "" || opt.KeyString != "") && (opt.CertFile == "" && opt.CertString == "") {
		resp.Diagnostics.AddError(
			"Incomplete Key Configuration",
			"A key is provided but no certificate is configured. Both cert and key must be provided for mTLS.",
		)
		return
	}

	// Handle OAuth client credentials if provided
	if data.OAuthClientCreds != nil {
		var oauthScopes []string
		if !data.OAuthClientCreds.OAuthScopes.IsNull() && !data.OAuthClientCreds.OAuthScopes.IsUnknown() {
			diags := data.OAuthClientCreds.OAuthScopes.ElementsAs(ctx, &oauthScopes, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		opt.OAuthClientID = existingOrEnvOrDefaultString(&resp.Diagnostics, "oauth_client_id", data.OAuthClientCreds.OAuthClientID, "REST_API_OAUTH_CLIENT_ID", "", true)
		opt.OAuthClientSecret = existingOrEnvOrDefaultString(&resp.Diagnostics, "oauth_client_secret", data.OAuthClientCreds.OAuthClientSecret, "REST_API_OAUTH_CLIENT_SECRET", "", true)
		opt.OAuthTokenURL = existingOrEnvOrDefaultString(&resp.Diagnostics, "oauth_token_url", data.OAuthClientCreds.OAuthTokenEndpoint, "REST_API_OAUTH_TOKEN_URL", "", true)
		opt.OAuthScopes = oauthScopes

		if !data.OAuthClientCreds.EndpointParams.IsNull() && !data.OAuthClientCreds.EndpointParams.IsUnknown() {
			var endpointParams map[string]string
			diags := data.OAuthClientCreds.EndpointParams.ElementsAs(ctx, &endpointParams, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			setVals := url.Values{}
			for k, v := range endpointParams {
				setVals.Add(k, v)
			}
			opt.OAuthEndpointParams = setVals
		}
	}

	// Final check for config errors
	if resp.Diagnostics.HasError() {
		return
	}

	// If URI is empty/unknown (depends on another resource), we'll defer client creation
	// until a resource/datasource calls GetClient()
	var client *apiclient.APIClient
	var err error

	if opt.URI != "" {
		client, err = apiclient.NewAPIClient(opt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create REST API Client",
				fmt.Sprintf("An unexpected error was encountered trying to create the REST API client. "+
					"Please verify your configuration settings are correct. Error: %s", err.Error()),
			)
			return
		}
	}

	// If a test_path is provided, issue a read_method request to it
	if tmp := existingOrEnvOrDefaultString(&resp.Diagnostics, "test_path", data.TestPath, "REST_API_TEST_PATH", "", false); tmp != "" {
		if client == nil {
			resp.Diagnostics.AddWarning(
				"Cannot Test Connection",
				"A test_path is configured but the provider URI is not yet known. The provider cannot test the connection. "+
					"Please ensure the provider URI is statically configured or depends on resources that have already been created.",
			)
		} else {
			_, _, err := client.SendRequest(ctx, opt.ReadMethod, tmp, "", opt.Debug)
			if err != nil {
				resp.Diagnostics.AddError(
					"Test Request Failed",
					fmt.Sprintf("A test request to %v after setting up the provider did not return an OK response - is your configuration correct? %v", tmp, err),
				)
			}
		}
	}

	providerData := &ProviderData{
		client: client,
		opts:   opt,
	}
	resp.ResourceData = providerData
	resp.DataSourceData = providerData
}

func (p *RestAPIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRestAPIObjectDataSource,
	}
}

func (p *RestAPIProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRestAPIObjectResource,
	}
}

func (p *RestAPIProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *RestAPIProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RestAPIProvider{
			version: version,
		}
	}
}
