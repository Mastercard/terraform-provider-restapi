provider "restapi" {
  uri = "https://example.internal"
}

# restapi_setting manages an existing singleton endpoint.
# On create it snapshots the current server state, then applies `data`.
# On destroy it restores only the managed keys from that snapshot.
resource "restapi_setting" "app_config" {
  read_path   = "/setting"
  update_path = "/setting"

  data = jsonencode({
    feature_flags = {
      enable_beta = true
      audit_mode  = "strict"
    }
    retention_days = 30
  })

  ignore_server_additions = true
}
