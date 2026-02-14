provider "restapi" {
  uri                  = "https://api.url.com"
  write_returns_object = true
  debug                = true

  headers = {
    "X-Auth-Token" = var.AUTH_TOKEN,
    "Content-Type" = "application/json"
  }

  create_method  = "PUT"
  update_method  = "PUT"
  destroy_method = "PUT"

  # Optional: Extract objects from wrapped GET responses
  # Use this if your API returns responses like: {"result": {...}, "success": true}
  # Supports nested paths: "data/items", "response/resource"
  # read_object_key = "result"
}