#This example demonstrates how you can pass external parameters into Terraform for use
# in this provider. For example, if you would like to pass a secret authorization token
# in from the environment, you could execute the following shell commands:
#export TF_VAR_SECRET_TOKEN=$(some_special_thing_to_get_credential)
#terraform apply
#
#This will cause the provider to send an HTTP request to the server with the following headers
# X-Internal-Client: abc123
# Authorization: <your external value>

variable "SECRET_TOKEN" {
  type = string
}

provider "restapi" {
  alias                = "restapi_headers"
  uri                  = "http://127.0.0.1:8080/"
  debug                = true
  write_returns_object = true

  headers = {
    X-Internal-Client = "abc123"
    Authorization = var.SECRET_TOKEN
  }
}

resource "restapi_object" "Foo2" {
  provider = restapi.restapi_headers
  path = "/api/objects"
  data = "{ \"id\": \"55555\", \"first\": \"Foo\", \"last\": \"Bar\" }"
}
