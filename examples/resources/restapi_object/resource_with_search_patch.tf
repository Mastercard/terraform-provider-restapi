# Example: Using read_search with search_patch to transform API responses
#
# When an API returns objects in a wrapped structure or with extra metadata,
# you can use search_patch to transform the response to match your desired state.

resource "restapi_object" "user_with_search_patch" {
  path = "/api/users"
  data = jsonencode({
    id    = "user123"
    name  = "John Doe"
    email = "john@example.com"
  })

  # Search for the object by email and transform the response
  read_search = {
    search_key   = "email"
    search_value = "john@example.com"

    # If API returns: {"data": {"id": "user123", "name": "John Doe", ...}, "metadata": {...}}
    # Transform it to: {"id": "user123", "name": "John Doe", ...}
    search_patch = jsonencode([
      { op = "copy", from = "/data/id", path = "/id" },
      { op = "copy", from = "/data/name", path = "/name" },
      { op = "copy", from = "/data/email", path = "/email" },
      { op = "remove", path = "/data" },
      { op = "remove", path = "/metadata" }
    ])
  }
}

# Example: Remove server-generated fields from API responses
resource "restapi_object" "clean_response" {
  path = "/api/resources"
  data = jsonencode({
    id   = "resource456"
    name = "My Resource"
  })

  read_search = {
    search_key   = "id"
    search_value = "{id}" # {id} placeholder is replaced with the object's ID

    # Remove fields that the server adds but Terraform shouldn't manage
    search_patch = jsonencode([
      { op = "remove", path = "/createdAt" },
      { op = "remove", path = "/updatedAt" },
      { op = "remove", path = "/metadata" }
    ])
  }
}
