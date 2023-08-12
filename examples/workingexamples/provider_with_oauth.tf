provider "restapi" {
  alias                = "restapi_oauth"
  uri                  = "http://127.0.0.1:8080/"
  debug                = true
  write_returns_object = true

  oauth_client_credentials {
    oauth_client_id      = "example"
    oauth_client_secret  = "example"
    oauth_token_endpoint = "https://example.com/tokenendpoint"
    oauth_scopes         = ["openid"]
    endpoint_params      = <<ENDPOINT_PARAMS
    {
      "resource": "http://127.0.0.1/"
    }
    ENDPOINT_PARAMS
  }
}
