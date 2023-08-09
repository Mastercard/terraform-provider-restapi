resource "restapi_object" "Foo2" {
  provider = restapi.restapi_headers
  path = "/api/objects"
  data = "{ \"id\": \"55555\", \"first\": \"Foo\", \"last\": \"Bar\" }"
}
