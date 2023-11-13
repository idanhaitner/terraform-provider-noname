terraform {
  required_providers {
    noname = {
      source = "hashicorp.com/edu/noname"
      # For local development,
      # install the provider on local computer by running `make install` from the root of the repo,
      # and uncomment the version below
      # version = "9999.99.99"
    }
  }
}

# Configure the AWS Provider
provider "noname" {
}

resource "noname_api_gateway" "default" {
  rest_api_id = "yx8upwzuqf"
  stage_name  = "access-log"
  description = "my name"
}
