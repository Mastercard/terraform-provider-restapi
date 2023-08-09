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