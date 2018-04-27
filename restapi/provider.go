package restapi

import (
 "github.com/hashicorp/terraform/helper/schema"
 "github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
  return &schema.Provider{
    Schema: map[string]*schema.Schema{
      "uri": &schema.Schema{
        Type: schema.TypeString,
        Required: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_URI", nil),
        Description: "URI of the REST API endpoint. This serves as the base of all requests.",
      },
      "insecure": &schema.Schema{
        Type: schema.TypeBool,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_INSECURE", nil),
        Description: "When using https, this disables TLS verification of the host.",
      },
      "username": &schema.Schema{
        Type: schema.TypeString,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_USERNAME", nil),
        Description: "When set, will use this username for BASIC auth to the API.",
      },
      "password": &schema.Schema{
        Type: schema.TypeString,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_PASSWORD", nil),
        Description: "When set, will use this password for BASIC auth to the API.",
      },
      "authorization_header": &schema.Schema{
        Type: schema.TypeString,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_AUTH_HEADER", nil),
        Description: "If the API does not support BASIC authentication, you can set the Authorization header contents to be sent in all requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token. This takes precedence over BASIC auth credentials.",
      },
      "timeout": &schema.Schema{
        Type: schema.TypeInt,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_TIMEOUT", 0),
        Description: "When set, will cause requests taking longer than this time (in seconds) to be aborted.",
      },
      "id_attribute": &schema.Schema{
        Type: schema.TypeString,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_ID_ATTRIBUTE", nil),
        Description: "When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to http://foo.com/bar/VALUE_OF_NAME",
      },
      "copy_keys": &schema.Schema{
        Type: schema.TypeList,
        Elem: &schema.Schema{Type: schema.TypeString},
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_PASSWORD", nil),
        Description: "When set, any PUT to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.",
      },
      "write_returns_object": &schema.Schema{
        Type: schema.TypeBool,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_WRO", nil),
        Description: "Set this when the API returns the object created on all write operations (POST, PUT). This is used by the provider to refresh internal data structures.",
      },
      "create_returns_object": &schema.Schema{
        Type: schema.TypeBool,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_CRO", nil),
        Description: "Set this when the API returns the object created only on creation operations (POST). This is used by the provider to refresh internal data structures.",
      },
      "debug": &schema.Schema{
        Type: schema.TypeBool,
        Optional: true,
        DefaultFunc: schema.EnvDefaultFunc("REST_API_DEBUG", nil),
        Description: "Enabling this will cause lots of debug information to be printed to STDOUT by the API client.",
      },
    },
    ResourcesMap: map[string]*schema.Resource{
      /* Could only get terraform to recognize this resource if
         the name began with the provider's name and had at least
	 one underscore. This is not documented anywhere I could find */
      "restapi_object": resourceRestApi(),
    },
    ConfigureFunc: configureProvider,
  }
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {

  /* As "data-safe" as terraform says it is, you'd think
     it would have already coaxed this to a slice FOR me */
  copy_keys := make([]string, 0)
  if i_copy_keys := d.Get("copy_keys"); i_copy_keys != nil {
    for _, v := range i_copy_keys.([]interface{}) {
      copy_keys = append(copy_keys, v.(string))
    }
  }

  return NewAPIClient(
    d.Get("uri").(string),
    d.Get("insecure").(bool),
    d.Get("username").(string),
    d.Get("password").(string),
    d.Get("authorization_header").(string),
    d.Get("timeout").(int),
    d.Get("id_attribute").(string),
    copy_keys,
    d.Get("write_returns_object").(bool),
    d.Get("create_returns_object").(bool),
    d.Get("debug").(bool),
  ), nil
}
