---
page_title: "restapi_object Resource - terraform-provider-restapi"
subcategory: ""
description: |-
  
---

# Resource `restapi_object`





## Schema

### Required

- **data** (String, Required) Valid JSON object that this provider will manage with the API server.
- **path** (String, Required) The API path on top of the base URL set in the provider that represents objects of this type on the API server.

### Optional

- **create_method** (String, Optional) Defaults to `create_method` set on the provider. Allows per-resource override of `create_method` (see `create_method` provider config documentation)
- **create_path** (String, Optional) Defaults to `path`. The API path that represents where to CREATE (POST) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object if the data contains the `id_attribute`.
- **debug** (Boolean, Optional) Whether to emit verbose debug output while working with the API object on the server.
- **destroy_data** (String, Optional) Valid JSON object to pass during to destroy requests.
- **destroy_method** (String, Optional) Defaults to `destroy_method` set on the provider. Allows per-resource override of `destroy_method` (see `destroy_method` provider config documentation)
- **destroy_path** (String, Optional) Defaults to `path/{id}`. The API path that represents where to DESTROY (DELETE) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.
- **force_new** (List of String, Optional) Any changes to these values will result in recreating the resource instead of updating.
- **id** (String, Optional) The ID of this resource.
- **id_attribute** (String, Optional) Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)
- **object_id** (String, Optional) Defaults to the id learned by the provider during normal operations and `id_attribute`. Allows you to set the id manually. This is used in conjunction with the `*_path` attributes.
- **query_string** (String, Optional) Query string to be included in the path
- **read_method** (String, Optional) Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)
- **read_path** (String, Optional) Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.
- **read_search** (Map of String, Optional) Custom search for `read_path`. This map will take `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)
- **update_data** (String, Optional) Valid JSON object to pass during to update requests.
- **update_method** (String, Optional) Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation)
- **update_path** (String, Optional) Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.

### Read-only

- **api_data** (Map of String, Read-only) After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).
- **api_response** (String, Read-only) The raw body of the HTTP response from the last read of the object.
- **create_response** (String, Read-only) The raw body of the HTTP response returned when creating the object.


