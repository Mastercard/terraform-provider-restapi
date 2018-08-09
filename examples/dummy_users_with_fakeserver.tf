# In this example, we are using the fakeserver available with this provider
# to create and manage imaginary users in our imaginary API server
# https://github.com/Mastercard/terraform-provider-restapi/tree/master/fakeserver

# To use this example fully, start up fakeserver and run this command
# curl 127.0.0.1:8080/api/objects -X POST -d '{ "id": "8877", "first": "John", "last": "Doe" }'
#
# After running terraform apply, you will now see three objects in the API server:
# curl 127.0.0.1:8080/api/objects | jq

provider "restapi" {
  uri                  = "http://127.0.0.1:8080/"
  debug                = true
  write_returns_object = true
}

# This will make information about the user named "John Doe" available by finding him by first name
data "restapi_object" "John" {
  path = "/api/objects"
  search_key = "first"
  search_value = "John"
}

# This will ADD the user named "Foo" as a managed resource
resource "restapi_object" "Foo" {
  path = "/api/objects"
  data = "{ \"id\": \"1234\", \"first\": \"Foo\", \"last\": \"Bar\" }"
}

#Congrats to Jane and John! They got married. Give them the same last name by using variable interpolation
resource "restapi_object" "Jane" {
  path = "/api/objects"
  data = "{ \"id\": \"7788\", \"first\": \"Jane\", \"last\": \"${data.restapi_object.John.api_data.last}\" }"
}

