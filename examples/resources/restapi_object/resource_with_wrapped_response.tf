# Example: API that wraps GET responses in an envelope structure
# POST/PUT sends: {"name": "foo", "config": {...}}
# GET returns: {"result": {"name": "foo", "config": {...}}, "success": true}
#
# Without read_object_key, this causes false drift because the provider
# compares the wrapped response to the unwrapped request data.

resource "restapi_object" "example_wrapped" {
  path = "/api/objects"

  # Extract the actual object from the "result" wrapper before state storage
  read_object_key = "result"

  data = jsonencode({
    name   = "my-resource"
    config = {
      setting = "value"
    }
  })
}

# Example: Nested wrapper path
# GET returns: {"response": {"data": {"id": "123", "value": "bar"}}, "meta": {...}}
resource "restapi_object" "example_nested" {
  path = "/api/objects"

  # Use slash-delimited path to extract from nested structure
  read_object_key = "response/data"

  data = jsonencode({
    id    = "123"
    value = "bar"
  })
}

# Example: Provider-level default with resource override
provider "restapi" {
  uri = "https://api.example.com"

  # Set default extraction key for all resources
  read_object_key = "result"
}

resource "restapi_object" "uses_provider_default" {
  path = "/api/objects"
  # This resource will use "result" from provider config

  data = jsonencode({
    name = "resource-one"
  })
}

resource "restapi_object" "overrides_provider" {
  path = "/api/other"

  # Override provider-level setting for this specific resource
  read_object_key = "data"

  data = jsonencode({
    name = "resource-two"
  })
}
