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

&nbsp;

## Provider configuration
- `uri` (string, required): URI of the REST API endpoint. This serves as the base of all requests. Example: `https://myapi.env.local/api/v1`.
- `insecure` (boolean, optional): When using https, this disables TLS verification of the host.
- `username` (string, optional): When set, will use this username for BASIC auth to the API.
- `password` (string, optional): When set, will use this password for BASIC auth to the API.
- `authorization_header` (string, optional): If the API does not support BASIC authentication, you can set the Authorization header contents to be sent in all requests. This is useful if you want to use a script via the `external` provider or provide a pre-approved token. This takes precedence over BASIC auth credentials.
- `timeout` (integer, optional): When set, will cause requests taking longer than this time (in seconds) to be aborted. Default is `0` which means no timeout is set.
- `id_attribute` (string, optional): When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to `http://foo.com/bar/VALUE_OF_NAME`.
- `copy_keys` (array of strings, optional): When set, any `PUT` to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.
- `write_returns_object` (boolean, optional): Set this when the API returns the object created on all write operations (`POST`, `PUT`). This is used by the provider to refresh internal data structures.
- `create_returns_object` (boolean, optional): Set this when the API returns the object created only on creation operations (`POST`). This is used by the provider to refresh internal data structures.
- `debug` (boolean, optional): Enabling this will cause lots of debug information to be printed to STDOUT by the API client. This can be gathered by setting `TF_LOG=1` environment variable.

&nbsp;

## `restapi` resource configuration
- `path` (string, required): The API path on top of the base URL set in the provider that represents objects of this type on the API server.
- `data` (string, required): Valid JSON data that this provider will manage with the API server. This should represent the whole API object that you want to create. The provider's information.
- `debug` (boolean, optional): Whether to emit verbose debug output while working with the API object on the server. This can be gathered by setting `TF_LOG=1` environment variable.

This provider also exports the following parameters:
- `id`: The ID of the object that is being managed.
- `api_data`: After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).

&nbsp;

## `restapi` datasource configuration
- `path` (string, required): The API path on top of the base URL set in the provider that represents objects of this type on the API server.
- `search_key` (string, required): When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'.
- `search_value` (string, required): The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.
- `results_key` (string, required): When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are to be used exactly as-is.
- `debug` (boolean, optional): Whether to emit verbose debug output while working with the API object on the server. This can be gathered by setting `TF_LOG=1` environment variable.

This provider also exports the following parameters:
- `id`: The native ID of the API object as the API server recognizes it.
- `api_data`: After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).
