[![Build Status](https://travis-ci.com/burbon/terraform-provider-restapi.svg?branch=master)](https://travis-ci.com/burbon/terraform-provider-restapi)
[![Coverage Status](https://coveralls.io/repos/github/burbon/terraform-provider-restapi/badge.svg?branch=master)](https://coveralls.io/github/burbon/terraform-provider-restapi?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/burbon/terraform-provider-restapi)](https://goreportcard.com/report/github.com/burbon/terraform-provider-restapi)
# Terraform provider for generic REST APIs

This terraform provider allows you to interact with APIs that may not yet have a first-class provider available.

There are a few requirements about how the API must work for this provider to be able to do its thing:
* The API is expected to support the following HTTP methods:
    * POST: create an object
    * GET: read an object
    * PUT: update an object
    * DELETE: remove an object
* An "object" in the API has a unique identifier the API will return
* Objects live under a distinct path such that for the path `/api/v1/things`...
    * POST on `/api/v1/things` creates a new object
    * GET, PUT and DELETE on `/api/v1/things/{id}` manages an existing object

Have a look at the [examples directory](examples) for some use cases

&nbsp;

## Provider configuration
- `uri` (string, required): URI of the REST API endpoint. This serves as the base of all requests. Example: `https://myapi.env.local/api/v1`. This can also be set with the environment variable `REST_API_URI`.
- `insecure` (boolean, optional): When using https, this disables TLS verification of the host. This can also be set with the environment variable `REST_API_INSECURE`.
- `username` (string, optional): When set, will use this username for BASIC auth to the API. This can also be set with the environment variable `REST_API_USERNAME`.
- `password` (string, optional): When set, will use this password for BASIC auth to the API. This can also be set with the environment variable `REST_API_PASSWORD`.
- `headers` (hash of strings, optional): A map of header names and values to set on all outbound requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token or change Content-Type from `application/json`. If `username` and `password` are set and Authorization is one of the headers defined here, the BASIC auth credentials take precedence.
- `use_cookies` (boolean, optional): Enable cookie jar to persist session. This can also be set with the environment variable `REST_API_USE_COOKIES`.
- `timeout` (integer, optional): When set, will cause requests taking longer than this time (in seconds) to be aborted. Default is `0` which means no timeout is set. This can also be set with the environment variable `REST_API_TIMEOUT`.
- `id_attribute` (string, optional): Defaults to `id`. When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to http://foo.com/bar/VALUE_OF_NAME. This value may also be a '/'-delimeted path to the id attribute if it is multple levels deep in the data (such as `attributes/id` in the case of an object `{ \"attributes\": { \"id\": 1234 }, \"config\": { \"name\": \"foo\", \"something\": \"bar\"}}`. Lists are also supported, f.e. `attributes/items/0/id` in case of an object `{ \"attributes\": { \"items\": [{"id": 1234}] } }`. This can also be set with the environment variable `REST_API_ID_ATTRIBUTE`.
- `create_method` (string, optional): Defaults to `POST`. The HTTP method used to CREATE objects of this type on the API server. This can also be set with the environment variable `REST_API_CREATE_METHOD`.
- `read_method` (string, optional): Defaults to `GET`. The HTTP method used to READ objects of this type on the API server. This can also be set with the environment variable `REST_API_READ_METHOD`.
- `update_method` (string, optional): Defaults to `PUT`. The HTTP method used to UPDATE objects of this type on the API server. This can also be set with the environment variable `REST_API_UPDATE_METHOD`.
- `destroy_method` (string, optional): Defaults to `DELETE`. The HTTP method used to DELETE objects of this type on the API server. This can also be set with the environment variable `REST_API_DESTROY_METHOD`.
- `copy_keys` (array of strings, optional): When set, any `PUT` to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.
- `write_returns_object` (boolean, optional): Set this when the API returns the object created on all write operations (`POST`, `PUT`). This is used by the provider to refresh internal data structures. This can also be set with the environment variable `REST_API_WRO`.
- `create_returns_object` (boolean, optional): Set this when the API returns the object created only on creation operations (`POST`). This is used by the provider to refresh internal data structures. This can also be set with the environment variable `REST_API_CRO`.
- `xssi_prefix` (boolean, optional): Trim the xssi prefix from response string, if present, before parsing. This can also be set with the environment variable `REST_API_XSSI_PREFIX`.
- `rate_limit` (float, optional): Set this to limit the number of requests per second made to the API.
- `debug` (boolean, optional): Enabling this will cause lots of debug information to be printed to STDOUT by the API client. This can be gathered by setting `TF_LOG=1` environment variable. This can also be set with the environment variable `REST_API_DEBUG`.

&nbsp;

## `restapi` resource configuration
- `path` (string, required): The API path on top of the base URL set in the provider that represents objects of this type on the API server.
- `create_path` (string, optional): Defaults to `path`. The API path that represents where to CREATE (POST) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object if the data contains the `id_attribute`.
- `read_path` (string, optional): Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.
- `update_path` (string, optional): Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.
- `create_method` (string, optional): Defaults to `create_method` set on the provider. Allows per-resource override of `create_method` (see `create_method` provider config documentation).
- `read_method` (string, optional): Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation).
- `update_method` (string, optional): Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation).
- `destroy_method` (string, optional): Defaults to `destroy_method` set on the provider. Allows per-resource override of `destroy_method` (see `destroy_method` provider config documentation).
- `destroy_path` (string, optional): Defaults to `path/{id}`. The API path that represents where to DESTROY (DELETE) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.
- `id_attribute` (string, optional): Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation).
- `object_id` (string, optional): Defaults to the id learned by the provider during normal operations and `id_attribute`. Allows you to set the id manually. This is used in conjunction with the `*_path` attributes.
- `data` (string, required): Valid JSON data that this provider will manage with the API server. This should represent the whole API object that you want to create. The provider's information.
- `force_new` (array of strings, optional): Any changes to these values will result in recreating the resource instead of updating.
- `debug` (boolean, optional): Whether to emit verbose debug output while working with the API object on the server. This can be gathered by setting `TF_LOG=1` environment variable.
- `read_search` (map, optional): Custom search for `read_path`. This map will take `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation).

This provider also exports the following parameters:
- `id`: The ID of the object that is being managed.
- `api_data`: After data from the API server is read, this map will include k/v pairs usable in other Terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).
- `api_response`: Contains the raw JSON response read back from the API server. Can be parsed with [`jsondecode`](https://www.terraform.io/docs/configuration/functions/jsondecode.html) to allow access to deeply nested data.
- `create_response`: Contains the raw JSON response from the initial object creation - use when an API only returns required data during create. Can be parsed with [`jsondecode`](https://www.terraform.io/docs/configuration/functions/jsondecode.html) to allow access to deeply nested data.

Note that the `*_path` elements are for very specific use cases where one might initially create an object in one location, but read/update/delete it on another path. For this reason, they allow for substitution to be done by the provider internally by injecting the `id` somewhere along the path. This is similar to terraform's substitution syntax in the form of `${variable.name}`, but must be done within the provider due to structure. The only substitution available is to replace the string `{id}` with the internal (terraform) `id` of the object as learned by the `id_attribute`.

By default data isn't considered as sensitive. If you want to hide `data's` value in output you would need to set environment variable `API_DATA_IS_SENSITIVE="true"`.
&nbsp;

## `restapi` datasource configuration
- `path` (string, required): The API path on top of the base URL set in the provider that represents objects of this type on the API server.
- `query_string` (string, optional): An optional query string to send when performing the search.
- `search_key` (string, required): When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'.
- `search_value` (string, required): The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.
- `results_key` (string, required): When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is
- `id_attribute` (string, optional): Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation).
- `debug` (boolean, optional): Whether to emit verbose debug output while working with the API object on the server. This can be gathered by setting `TF_LOG=1` environment variable.

This provider also exports the following parameters:
- `id`: The native ID of the API object as the API server recognizes it.
- `api_data`: After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).
- `api_response`: Contains the raw JSON response read back from the API server. Can be parsed with [`jsondecode`](https://www.terraform.io/docs/configuration/functions/jsondecode.html) to allow access to deeply nested data.

## Importing existing resources
This provider supports importing existing resources into the terraform state. Import is done according to the various provider/resource configuation settings to contact the API server and obtain data. That is: if a custom read method, path, or id attribute is defined, the provider will honor those settings to pull data in.

To import data:
`terraform import restapi.Name /path/to/resource`

See a concrete example [here](examples/dummy_users_with_fakeserver.tf)
&nbsp;

## Installation
There are two standard methods of installing this provider detailed [in Terraform's documentation](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins). You can place the file in the directory of your .tf file in `terraform.d/plugins/{OS}_{ARCH}/` or place it in your home directory at `~/.terraform.d/plugins/{OS}_{ARCH}/`

The released binaries are named `terraform-provider-restapi_vX.Y.Z-{OS}-{ARCH}` so you know which binary to install. You *may* need to rename the binary you use during installation to just `terraform-provider-restapi_vX.Y.Z`.

Once downloaded, be sure to make the plugin executable by running `chmod +x terraform-provider-restapi_vX.Y.Z-{OS}-{ARCH}`.

&nbsp;

## Contributing
Pull requests are always welcome! Please be sure the following things are taken care of with your pull request:
* `go fmt` is run before pushing
* Be sure to add a test case for new functionality (or explain why this cannot be done)
* Run the `scripts/test.sh` script to be sure everything works
* Ensure new attributes can also be set by environment variables

#### Development environment requirements
* [Golang](https://golang.org/dl/) v1.11 or newer is installed and `go` is in your path
* [Terraform](https://www.terraform.io/downloads.html) is installed and `terraform` is in your path

To make development easy, you can use the Docker image [druggeri/tdk](https://hub.docker.com/r/druggeri/tdk) as a development environment:
```
docker run -it --name tdk --rm -v "$HOME/go":/root/go druggeri/tdk
go get github.com/Mastercard/terraform-provider-restapi
cd ~/go/src/github.com/Mastercard/terraform-provider-restapi
#Hack hack hack
```
