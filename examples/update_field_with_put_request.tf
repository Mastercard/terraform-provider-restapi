# This example demonstrate how you can use this provider to update one or more values
# with a PUT method like you are doing it with curl.
# To accomplish this you will need to update the default methods for create and destroy, ensure they are set to PUT.
# You will also need to modifiy the path in the "restapi_object" resource to make it conform for a PUT request.
# (Like you can see in the logs when a PUT request is called the provider add a {id} to the default path)
# This example may work for each HTTP REQUEST METHOD if you adapt correctly the path and the "method" fields.

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
}

resource "restapi_object" "put_request" {
  path         = "/instance/v1/zones/fr-par-1/security_groups/{id}"
  id_attribute = var.ID
  object_id    = var.ID
  data         = "{ \"id\": \"${var.ID}\",\"FIELD_TO_UPDATE\": update_1 }"
}
