# Demonstrates the issue when API returns additional fields (like IBM Cloud)
#
# Steps to reproduce:
# 1. Start the test server: go run server.go
# 2. Run: terraform init && terraform apply -auto-approve
# 3. Run: terraform plan  <-- This should show the issue
#
# Workaround: Set ignore_server_additions = true (uncomment below)

terraform {
  required_providers {
    restapi = {
      source  = "Mastercard/restapi"
      version = ">= 3.0.0-rc1"
    }
  }
}

provider "restapi" {
  uri                  = "http://localhost:8080"
  write_returns_object = true
}

# Simulates IBM Cloud pattern: PATCH to a resource path with ID
# User only specifies nested "parameters" object
# But the server returns 30+ additional fields
resource "restapi_object" "example" {
  path = "/api/objects/my-resource-id"

  data = jsonencode({
    parameters = {
      enable_feature = true
      config = {
        setting_a = "value1"
        setting_b = "value2"
      }
    }
  })

  create_method  = "PATCH"
  update_method  = "PATCH"
  destroy_method = "PATCH"

  # WORKAROUND: Uncomment this line to fix the error (requires PR #345 fix)
  # ignore_server_additions = true
}

output "api_response" {
  description = "The full API response (notice the extra fields added by server)"
  value       = restapi_object.example.api_response
}
