provider "restapi" {
    alias = "restapi_oauth_env"
    uri = "https://graph.microsoft.com/beta/"
    write_returns_object = true 
    debug = true

    oauth_client_credentials {
        oauth_client_id_environment_variable = "ARM_CLIENT_ID"
        oauth_client_secret_environment_variable = "ARM_CLIENT_SECRET"
        oauth_token_endpoint = "https://login.microsoft.com/${var.tenantId}/oauth2/v2.0/token"
        oauth_scopes = ["https://graph.microsoft.com/.default"]
        endpoint_params = {"grant_type"="client_credentials"}
    }

    headers = {
        "Content-Type" = "application/json"
    }

    create_method = "PUT"
    update_method = "PATCH"
    destroy_method = "DELETE"
}