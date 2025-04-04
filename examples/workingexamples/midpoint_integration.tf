# Midpoint integration example
# This demonstrates how to use the provider with Midpoint's REST API

provider "restapi" {
  uri                  = "http://midpoint-server:8080/midpoint/api"
  debug                = true
  write_returns_object = true
  
  # Use PATCH for updates to enable Midpoint's ObjectModificationType functionality
  update_method        = "PATCH"
  
  # Basic authentication for the Midpoint server
  username             = "administrator"
  password             = "5ecr3t"
  
  # Headers for Midpoint API
  headers = {
    Content-Type = "application/json"
    Accept       = "application/json"
  }
}

# Example resource for a Midpoint user
resource "restapi_object" "user_example" {
  path = "/users"
  
  # Initial data for the user
  data = jsonencode({
    name        = "jsmith"
    givenName   = "John"
    familyName  = "Smith"
    emailAddress = "john.smith@example.com"
    enabled     = true
  })
  
  # Use the same PATCH method configured in the provider
  # or override it here if needed
  update_method = "PATCH"
}

# Later, you can change specific attributes and the provider
# will automatically detect changes and send only the modifications
# to the Midpoint server using the PATCH method with ObjectModificationType format.

# For example, if you change the data to:
# data = jsonencode({
#   name        = "jsmith"
#   givenName   = "John"
#   familyName  = "Smith"
#   emailAddress = "john.smith.new@example.com"  # This changed
#   enabled     = true
#   description = "Updated user"  # This is new
# })
# 
# The provider will send two PATCH requests:
# 1. To update emailAddress (modificationType=replace)
# 2. To add description (modificationType=add)