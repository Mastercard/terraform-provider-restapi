provider "restapi" {
  alias                = "restapi_ms_azure"
  uri                  = "https://management.azure.com/subscriptions/${var.subscription_id}/"
  debug                = true
  write_returns_object = true

  oauth_client_credentials {
      oauth_client_id       = "example"
      oauth_client_secret   = "example"
      oauth_token_endpoint  = "https://login.microsoftonline.com/${var.tenant_id}/oauth2/token"
      oauth_scopes          = ["openid"]
      oauth_endpoint_params = {
        resource = "https://management.core.windows.net"
      }
  }
}
