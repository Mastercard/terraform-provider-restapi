---
page_title: "restapi_object Data Source - terraform-provider-restapi"
subcategory: ""
description: |-
  
---

# Data Source `restapi_object`





## Schema

### Required

- **path** (String, Required) The API path on top of the base URL set in the provider that represents objects of this type on the API server.
- **search_key** (String, Required) When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'. Similar to results_key, the value may be in the format of 'field/field/field' to search for data deeper in the returned object.
- **search_value** (String, Required) The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.

### Optional

- **debug** (Boolean, Optional) Whether to emit verbose debug output while working with the API object on the server.
- **id** (String, Optional) The ID of this resource.
- **id_attribute** (String, Optional) Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)
- **query_string** (String, Optional) An optional query string to send when performing the search.
- **read_query_string** (String, Optional) Defaults to `query_string` set on data source. This key allows setting a different or empty query string for reading the object.
- **results_key** (String, Optional) When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is.
- **search_path** (String, Optional) The API path on top of the base URL set in the provider that represents the location to search for objects of this type on the API server. If not set, defaults to the value of path.

### Read-only

- **api_data** (Map of String, Read-only) After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).
- **api_response** (String, Read-only) The raw body of the HTTP response from the last read of the object.


