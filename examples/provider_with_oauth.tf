provider "restapi" {
  uri                  = "http://127.0.0.1:8080/"
  debug                = true
  write_returns_object = true

  oauth_client_credentials {
      client_id = "example"
      client_secret = "example"
      token_endpoint = "https://exmaple.com/tokenendpoint"
      scopes = ["openid"]
  }
}