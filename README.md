[![Build Status](https://travis-ci.com/burbon/terraform-provider-restapi.svg?branch=master)](https://travis-ci.com/burbon/terraform-provider-restapi)
[![Coverage Status](https://coveralls.io/repos/github/burbon/terraform-provider-restapi/badge.svg?branch=master)](https://coveralls.io/github/burbon/terraform-provider-restapi?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/burbon/terraform-provider-restapi)](https://goreportcard.com/report/github.com/burbon/terraform-provider-restapi)
# Terraform provider for generic REST APIs

## Maintenance Note
This provider is largely feature-complete and in maintenance mode.
* It's not dead - it's just slow moving and updates must be done very carefully
* We encourage community participation with open issues for usage and remain welcoming of pull requests
* Code updates happen sporadically throughout the year, driven primarily by security fixes and PRs
* Because of the many API variations and flexibility of this provider, detailed per-API troubleshooting cannot be guaranteed

&nbsp;

## About This Provider
This terraform provider allows you to interact with APIs that may not yet have a first-class provider available by implementing a "dumb" REST API client.

This provider is essentially created to be a terraform-wrapped `cURL` client. Because of this, you need to know quite a bit about the API you are interacting with as opposed to full-featured terraform providers written with a specific API in mind.

There are a few requirements about how the API must work for this provider to be able to do its thing:
* The API is expected to support the following HTTP methods:
    * POST: create an object
    * GET: read an object
    * PUT: update an object
    * DELETE: remove an object
* An "object" in the API has a unique identifier the API will return
* Objects live under a distinct path such that for the path `/api/v1/things`...
    * POST on `/api/v1/things` creates a new object
    * GET, PUT and DELETE on `/api/v1/things/{id}` manages an existing object

Have a look at the [examples directory](examples) for some use cases.

&nbsp;

## Provider Documentation
This provider has only a few moving components, but LOTS of configurable parameters:
* [provider documentation](https://registry.terraform.io/providers/Mastercard/restapi/latest/docs)
* [restapi_object resource documentation](https://registry.terraform.io/providers/Mastercard/restapi/latest/docs/resources/object)
* [restapi_object datasource documentation](https://registry.terraform.io/providers/Mastercard/restapi/latest/docs/data-sources/object)

&nbsp;

## Usage
* Try to set as few parameters as possible to begin with. The more complicated the configuration gets, the more difficult troubleshooting can become.
* Play with the [fakeserver cli tool](fakeservercli/) (included in releases) to get a feel for how this API client is expected to work. Also see the [examples directory](examples) directory for some working use cases with fakeserver.
* By default, data isn't considered sensitive. If you want to hide the data this provider submits as well as the data returned by the API, you would need to set environment variable `API_DATA_IS_SENSITIVE=true`.
* The `*_path` elements are for very specific use cases where one might initially create an object in one location, but read/update/delete it on another path. For this reason, they allow for substitution to be done by the provider internally by injecting the `id` somewhere along the path. This is similar to terraform's substitution syntax in the form of `${variable.name}`, but must be done within the provider due to structure. The only substitution available is to replace the string `{id}` with the internal (terraform) `id` of the object as learned by the `id_attribute`.

&nbsp;

### Troubleshooting
Because this provider is just a terraform-wrapped `cURL`, the API details and the go implementation of this client are often leaked to you.
This means you, as the user, will have a bit more troubleshooting on your hands than would typically be required of a full-fledged provider if you experience issues.

Here are some tips for troubleshooting that may be helpful...

&nbsp;

#### Debug log
**Rely heavily on the debug log.** The debug log, enabled by setting the environment variable `TF_LOG=1` and enabling the `debug` parameter on the provider, is the best way to figure out what is happening.

If an unexpected error occurs, enable debug log and review the output:
* Does the API return an odd HTTP response code? This is common for bad requests to the API. Look closely at the HTTP request details.
* Does an unexpected golang 'unmarshaling' error occur? Take a look at the debug log and see if anything other than a hash (for resources) or an array (for the datasource) is being returned. For example, the provider cannot cope with cases where a JSON object is requested, but an array of JSON objects is returned.

&nbsp;

### Importing existing resources
This provider supports importing existing resources into the terraform state. Import is done according to the various provider/resource configuation settings to contact the API server and obtain data. That is: if a custom read method, path, or id attribute is defined, the provider will honor those settings to pull data in.

To import data:
`terraform import restapi.Name /path/to/resource`.

See a concrete example [here](examples/dummy_users_with_fakeserver.tf).

&nbsp;

## Installation
There are two standard methods of installing this provider detailed [in Terraform's documentation](https://www.terraform.io/docs/configuration/providers.html#third-party-plugins). You can place the file in the directory of your .tf file in `terraform.d/plugins/{OS}_{ARCH}/` or place it in your home directory at `~/.terraform.d/plugins/{OS}_{ARCH}/`.

The released binaries are named `terraform-provider-restapi_vX.Y.Z-{OS}-{ARCH}` so you know which binary to install. You *may* need to rename the binary you use during installation to just `terraform-provider-restapi_vX.Y.Z`.

Once downloaded, be sure to make the plugin executable by running `chmod +x terraform-provider-restapi_vX.Y.Z-{OS}-{ARCH}`.

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

To make development easy, you can use the Docker image [druggeri/tdk](https://hub.docker.com/r/druggeri/tdk) as a development environment:
```
docker run -it --name tdk --rm -v "$HOME/go":/root/go druggeri/tdk
go get github.com/Mastercard/terraform-provider-restapi
cd ~/go/src/github.com/Mastercard/terraform-provider-restapi
#Hack hack hack
```
