package restapi

import (
	"fmt"
	"math"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

/*Provider implements the REST API provider*/
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"uri": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_URI", nil),
				Description: "URI of the REST API endpoint. This serves as the base of all requests.",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_INSECURE", nil),
				Description: "When using https, this disables TLS verification of the host.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_USERNAME", nil),
				Description: "When set, will use this username for BASIC auth to the API.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_PASSWORD", nil),
				Description: "When set, will use this password for BASIC auth to the API.",
			},
			"bearer_env_var_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_BEARER_ENV_VAR_NAME", nil),
				Description: "When set, will use the value of this env var for as bearer for auth to the API.",
			},
			"headers": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				Description: "A map of header names and values to set on all outbound requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token or change Content-Type from `application/json`. If `username` and `password` are set and Authorization is one of the headers defined here, the BASIC auth credentials take precedence.",
			},
			"use_cookies": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_USE_COOKIES", nil),
				Description: "Enable cookie jar to persist session.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_TIMEOUT", 0),
				Description: "When set, will cause requests taking longer than this time (in seconds) to be aborted.",
			},
			"id_attribute": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_ID_ATTRIBUTE", nil),
				Description: "When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to http://foo.com/bar/VALUE_OF_NAME. This value may also be a '/'-delimeted path to the id attribute if it is multple levels deep in the data (such as `attributes/id` in the case of an object `{ \"attributes\": { \"id\": 1234 }, \"config\": { \"name\": \"foo\", \"something\": \"bar\"}}`",
			},
			"create_method": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_CREATE_METHOD", nil),
				Description: "Defaults to `POST`. The HTTP method used to CREATE objects of this type on the API server.",
				Optional:    true,
			},
			"read_method": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_READ_METHOD", nil),
				Description: "Defaults to `GET`. The HTTP method used to READ objects of this type on the API server.",
				Optional:    true,
			},
			"update_method": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_UPDATE_METHOD", nil),
				Description: "Defaults to `PUT`. The HTTP method used to UPDATE objects of this type on the API server.",
				Optional:    true,
			},
			"destroy_method": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_DESTROY_METHOD", nil),
				Description: "Defaults to `DELETE`. The HTTP method used to DELETE objects of this type on the API server.",
				Optional:    true,
			},
			"copy_keys": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "When set, any PUT to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.",
			},
			"write_returns_object": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_WRO", nil),
				Description: "Set this when the API returns the object created on all write operations (POST, PUT). This is used by the provider to refresh internal data structures.",
			},
			"create_returns_object": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_CRO", nil),
				Description: "Set this when the API returns the object created only on creation operations (POST). This is used by the provider to refresh internal data structures.",
			},
			"xssi_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_XSSI_PREFIX", nil),
				Description: "Trim the xssi prefix from response string, if present, before parsing.",
			},
			"rate_limit": {
				Type:        schema.TypeFloat,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_RATE_LIMIT", math.MaxFloat64),
				Description: "Set this to limit the number of requests per second made to the API.",
			},
			"test_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_TEST_PATH", nil),
				Description: "If set, the provider will issue a read_method request to this path after instantiation requiring a 200 OK response before proceeding. This is useful if your API provides a no-op endpoint that can signal if this provider is configured correctly. Response data will be ignored.",
			},
			"debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_DEBUG", nil),
				Description: "Enabling this will cause lots of debug information to be printed to STDOUT by the API client.",
			},
			"oauth_client_credentials": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Configuration for oauth client credential flow",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"oauth_client_id": {
							Type:        schema.TypeString,
							Description: "client id",
							Required:    true,
						},
						"oauth_client_secret": {
							Type:        schema.TypeString,
							Description: "client secret",
							Required:    true,
						},
						"oauth_token_endpoint": {
							Type:        schema.TypeString,
							Description: "oauth token endpoint",
							Required:    true,
						},
						"oauth_scopes": {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "scopes",
						},
						"endpoint_params": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Additional key/values to pass to the underlying Oauth client library (as EndpointParams)",
							Elem: &schema.Schema{
								Type: schema.TypeList,
								Elem: &schema.Schema{Type: schema.TypeString},
							},
						},
					},
				},
			},
			"cert_string": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_CERT_STRING", nil),
				Description: "When set with the key_string parameter, the provider will load a client certificate as a string for mTLS authentication.",
			},
			"key_string": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_KEY_STRING", nil),
				Description: "When set with the cert_string parameter, the provider will load a client certificate as a string for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
			},
			"cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_CERT_FILE", nil),
				Description: "When set with the key_file parameter, the provider will load a client certificate as a file for mTLS authentication.",
			},
			"key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("REST_API_KEY_FILE", nil),
				Description: "When set with the cert_file parameter, the provider will load a client certificate as a file for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			/* Could only get terraform to recognize this resource if
			         the name began with the provider's name and had at least
				 one underscore. This is not documented anywhere I could find */
			"restapi_object": resourceRestAPI(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"restapi_object": dataSourceRestAPI(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {

	/* As "data-safe" as terraform says it is, you'd think
	   it would have already coaxed this to a slice FOR me */
	copyKeys := make([]string, 0)
	if iCopyKeys := d.Get("copy_keys"); iCopyKeys != nil {
		for _, v := range iCopyKeys.([]interface{}) {
			copyKeys = append(copyKeys, v.(string))
		}
	}

	headers := make(map[string]string)
	if iHeaders := d.Get("headers"); iHeaders != nil {
		for k, v := range iHeaders.(map[string]interface{}) {
			headers[k] = v.(string)
		}
	}

	opt := &apiClientOpt{
		uri:                 d.Get("uri").(string),
		insecure:            d.Get("insecure").(bool),
		username:            d.Get("username").(string),
		password:            d.Get("password").(string),
		bearer:              os.Getenv(d.Get("bearer_env_var_name").(string)),
		headers:             headers,
		useCookies:          d.Get("use_cookies").(bool),
		timeout:             d.Get("timeout").(int),
		idAttribute:         d.Get("id_attribute").(string),
		copyKeys:            copyKeys,
		writeReturnsObject:  d.Get("write_returns_object").(bool),
		createReturnsObject: d.Get("create_returns_object").(bool),
		xssiPrefix:          d.Get("xssi_prefix").(string),
		rateLimit:           d.Get("rate_limit").(float64),
		debug:               d.Get("debug").(bool),
	}

	if v, ok := d.GetOk("create_method"); ok {
		opt.createMethod = v.(string)
	}
	if v, ok := d.GetOk("read_method"); ok {
		opt.readMethod = v.(string)
	}
	if v, ok := d.GetOk("update_method"); ok {
		opt.updateMethod = v.(string)
	}
	if v, ok := d.GetOk("destroy_method"); ok {
		opt.destroyMethod = v.(string)
	}
	if v, ok := d.GetOk("oauth_client_credentials"); ok {
		oauthConfig := v.([]interface{})[0].(map[string]interface{})

		opt.oauthClientID = oauthConfig["oauth_client_id"].(string)
		opt.oauthClientSecret = oauthConfig["oauth_client_secret"].(string)
		opt.oauthTokenURL = oauthConfig["oauth_token_endpoint"].(string)
		opt.oauthScopes = expandStringSet(oauthConfig["oauth_scopes"].([]interface{}))

		if tmp, ok := oauthConfig["endpoint_params"]; ok {
			m := tmp.(map[string]interface{})
			setVals := url.Values{}
			for k, vals := range m {
				for _, val := range vals.([]string) {
					setVals.Add(k, val)
				}
			}
			opt.oauthEndpointParams = setVals
		}
	}
	if v, ok := d.GetOk("cert_file"); ok {
		opt.certFile = v.(string)
	}
	if v, ok := d.GetOk("key_file"); ok {
		opt.keyFile = v.(string)
	}
	if v, ok := d.GetOk("cert_string"); ok {
		opt.certString = v.(string)
	}
	if v, ok := d.GetOk("key_string"); ok {
		opt.keyString = v.(string)
	}

	client, err := NewAPIClient(opt)

	if v, ok := d.GetOk("test_path"); ok {
		testPath := v.(string)
		_, err := client.sendRequest(client.readMethod, testPath, "")
		if err != nil {
			return client, fmt.Errorf("a test request to %v after setting up the provider did not return an OK response - is your configuration correct? %v", testPath, err)
		}
	}
	return client, err
}
