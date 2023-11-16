terraform {
  required_providers {
    noname = {
      source = "hashicorp.com/edu/noname"
    }
  }
}

provider "noname" {

}

resource "noname_api_gateway_integration" "api_gateway" {
  rest_api_ids = []
}
