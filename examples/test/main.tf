terraform {
  required_providers {
    yb = {
      version = "~> 0.1.0"
      source  = "terraform.yugabyte.com/platform/yugabyte-platform"
    }
  }
}

data "yb_customer_data" "customer" {
  api_token = "***REMOVED***"
}

provider "yb" {
  // these can be set as environment variables
  host   = "35.203.183.215:80"
}

resource "yb_storage_config_resource" "config" {
  connection_info {
    cuuid     = data.yb_customer_data.customer.cuuid
    api_token = data.yb_customer_data.customer.api_token
  }

  config_name = "hi"
  data = {
    random = "hi"
  }
  name = "bye"
}