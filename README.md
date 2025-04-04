# Terraform provider for Midpoint REST APIs

## About This Provider


This Provider is just a fork of Terraform RestAPI (https://github.com/Mastercard/terraform-provider-restapi) providing additional to work with Evolveum's Midpoint (https://github.com/Evolveum/midpoint) REST APIs (https://docs.evolveum.com/midpoint/reference/support-4.9/interfaces/rest/).

This makes possible to manage Midpoint's resources partially, without having to handle things like Shadow Link References. This is specially useful to keep multiple environments (production, pre-production, etc...) partially in sync.

Changes are currently focused around the modify operation, instead of sending the whole object to the API endpoint, the provider should calculate the changes from the state, and send the changes using the PATCH method, and the body should contain the changes using ObjectModificationType (form Midpoint). These are some examples:


Add attributes to the object:


```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "add",
      "path": "description",
      "value": "Description parameter modified via REST"
    }
  }
}
```

Remove attributes from the object:

```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "delete",
      "path": "description",
    }
  }
}
```

Replace attribute values:

```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "replace",
      "path": "description",
      "value": "Description parameter modified via REST. Changed"
    }
  }
}
```

If multiple modifications are required, multiple calls should be made to the API endpoint each with its own itemDelta, but only if the changes are in different 1st level branches of the object.


Have a look at the [examples directory](examples) for some use cases.

&nbsp;


## Provider Documentation

&nbsp;

## Usage
* For Midpoint integration, set `update_method` to `PATCH`. This enables the provider to calculate changes between the current and desired state and send them using Midpoint's ObjectModificationType format. See the [midpoint integration example](examples/workingexamples/midpoint_integration.tf) for details.

&nbsp;

### Importing existing resources
This provider supports importing existing resources into the terraform state. Import is done according to the various provider/resource configuation settings to contact the API server and obtain data. That is: if a custom read method, path, or id attribute is defined, the provider will honor those settings to pull data in.

To import data:
`terraform import midpoint-restapi.Name /path/to/resource`.

See a concrete example [here](examples/workingexamples/dummy_users_with_fakeserver.tf).

&nbsp;

## Installation
There are two standard methods of installing this provider detailed [in Terraform's documentation](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins). You can place the file in the directory of your .tf file in `terraform.d/plugins/{OS}_{ARCH}/` or place it in your home directory at `~/.terraform.d/plugins/{OS}_{ARCH}/`.

The released binaries are named `terraform-provider-midpoint-restapi_vX.Y.Z-{OS}-{ARCH}` so you know which binary to install. You *may* need to rename the binary you use during installation to just `terraform-provider-restapi_vX.Y.Z`.

Once downloaded, be sure to make the plugin executable by running `chmod +x terraform-provider-midpoint-restapi_vX.Y.Z-{OS}-{ARCH}`.

&nbsp;

## Contributing
Pull requests are always welcome! Please be sure the following things are taken care of with your pull request:
* `go fmt` is run before pushing
* Be sure to add a test case for new functionality (or explain why this cannot be done)
* Run the `scripts/test.sh` script to be sure everything works
* Ensure new attributes can also be set by environment variables

#### Development environment requirements:
* [Golang](https://golang.org/dl/) v1.11 or newer is installed and `go` is in your path
* [Terraform](https://www.terraform.io/downloads.html) is installed and `terraform` is in your path
