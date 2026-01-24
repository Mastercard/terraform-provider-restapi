# Server Additions Bug - "Invalid JSON String Value" Error

This example reproduces the "Invalid JSON String Value" error that occurs in v3.0.0 when a REST API returns additional fields beyond what the user configured.

## The Problem

When using the restapi provider with PATCH-based APIs:
1. User configures a resource with specific fields (e.g., `parameters`)
2. The API returns a response with many additional fields (timestamps, metadata, etc.)
3. On subsequent `terraform plan`, the provider fails with:

```
Error: Invalid JSON String Value

  with restapi_object.example,
  on main.tf line 30, in resource "restapi_object" "example":

A string value was provided that is not valid JSON string format (RFC 7159).

Given Value:
```

This is a v3.0.0 regression caused by stricter JSON validation with `jsontypes.Normalized`.

## Steps to Reproduce

1. **Start the test server** (in one terminal):
   ```bash
   go run server.go
   ```

2. **Initialize and apply** (in another terminal):
   ```bash
   terraform init
   terraform apply -auto-approve
   ```

3. **Run plan** - this fails with "Invalid JSON String Value":
   ```bash
   terraform plan
   ```

## Workaround

Set `ignore_server_additions = true`:

```hcl
resource "restapi_object" "example" {
  path = "/api/objects/my-resource-id"
  data = jsonencode({
    parameters = {
      enable_feature = true
      config = { ... }
    }
  })

  create_method  = "PATCH"
  update_method  = "PATCH"
  destroy_method = "PATCH"

  ignore_server_additions = true  # <-- Add this
}
```

## Why This Happens

In v3.0.0, the provider uses `jsontypes.Normalized` for the `data` attribute, which has stricter validation than the plain strings used in v2.x.

When the API returns many additional fields:
1. These get stored in `state.Data`
2. During plan, Terraform compares `plan.Data` (user's config) with `state.Data` (full API response)
3. The validation fails with "Invalid JSON String Value"

## Test Server Behavior

The included `server.go` simulates a typical cloud API (like IBM Cloud Resource Controller) that:
- Accepts PATCH with user data
- Returns the object with 25+ additional server-generated fields:
  - `id`, `guid`, `crn`
  - `created_at`, `updated_at`, `created_by`, `updated_by`
  - `account_id`, `resource_group_id`, `resource_group_crn`
  - `state`, `type`, `sub_type`
  - `last_operation` (nested object)
  - `extensions` (deeply nested object with VPE config)
  - And more...

This is common behavior for real-world cloud APIs.

## Related Issues

- Issue #344: Original bug report for `ignore_server_additions` issues
- PR #345: Fix for `ignore_server_additions` implementation
