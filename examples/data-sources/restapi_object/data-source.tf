data "restapi_object" "John" {
  path = "/api/objects"
  search_key = "first"
  search_value = "John"
}