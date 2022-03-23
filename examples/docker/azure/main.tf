terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
    yb = {
      version = "~> 0.1.0"
      source  = "terraform.yugabyte.com/platform/yugabyte-platform"
    }
  }
}

provider "azurerm" {
  features {}
}

locals {
  dir = "/Users/stevendu/code/terraform-provider-yugabyte-anywhere/modules/resources"
}

module "azure_yb_anywhere" {
  source = "../../../modules/docker/azure"

  cluster_name        = "sdu-test-yugaware"
  ssh_user            = "sdu"
  region_name         = "westus2"
  subnet_name         = "***REMOVED***"
  vnet_name           = "***REMOVED***"
  vnet_resource_group = "yugabyte-rg"
  // files
  ssh_private_key = "/Users/stevendu/.ssh/yugaware-azure"
  ssh_public_key  = "/Users/stevendu/.ssh/yugaware-azure.pub"
}

provider "yb" {
  host = "${module.azure_yb_anywhere.public_ip}:80"
}

resource "yb_installation" "installation" {
  public_ip                 = module.azure_yb_anywhere.public_ip
  private_ip                = module.azure_yb_anywhere.private_ip
  ssh_user                  = "sdu"
  ssh_private_key           = file("/Users/stevendu/.ssh/yugaware-azure")
  replicated_config_file    = "${local.dir}/replicated.conf"
  replicated_license_file   = "/Users/stevendu/.yugabyte/yugabyte-dev.rli"
  application_settings_file = "${local.dir}/application_settings.conf"
}

resource "yb_customer_resource" "customer" {
  depends_on = [module.azure_yb_anywhere, yb_installation.installation]
  code       = "admin"
  email      = "sdu@yugabyte.com"
  name       = "sdu"
  password   = "Password1@"
}

resource "yb_cloud_provider" "gcp" {
  connection_info {
    cuuid     = yb_customer_resource.customer.cuuid
    api_token = yb_customer_resource.customer.api_token
  }

  code = "gcp"
  config = merge(
    { YB_FIREWALL_TAGS = "cluster-server" },
    jsondecode(file("/Users/stevendu/.yugabyte/yugabyte-gce.json"))
  )
  dest_vpc_id = "***REMOVED***"
  name        = "sdu-test-gcp-provider"
  regions {
    code = "us-west1"
    name = "us-west1"
  }
  ssh_port        = 54422
  air_gap_install = false
}

data "yb_provider_key" "gcp-key" {
  connection_info {
    cuuid     = yb_customer_resource.customer.cuuid
    api_token = yb_customer_resource.customer.api_token
  }

  provider_id = yb_cloud_provider.gcp.id
}

locals {
  region_list  = yb_cloud_provider.gcp.regions[*].uuid
  provider_id  = yb_cloud_provider.gcp.id
  provider_key = data.yb_provider_key.gcp-key.id
}

resource "yb_universe" "gcp_universe" {
  connection_info {
    cuuid     = yb_customer_resource.customer.cuuid
    api_token = yb_customer_resource.customer.api_token
  }

  depends_on = [yb_cloud_provider.gcp]
  clusters {
    cluster_type = "PRIMARY"
    user_intent {
      universe_name      = "sdu-test-gcp-universe-on-azure"
      provider_type      = "gcp"
      provider           = local.provider_id
      region_list        = local.region_list
      num_nodes          = 3
      replication_factor = 3
      instance_type      = "n1-standard-1"
      device_info {
        num_volumes  = 1
        volume_size  = 375
        storage_type = "Persistent"
      }
      assign_public_ip              = true
      use_time_sync                 = true
      enable_ysql                   = true
      enable_node_to_node_encrypt   = true
      enable_client_to_node_encrypt = true
      yb_software_version           = "2.13.1.0-b24"
      access_key_code               = local.provider_key
    }
  }
  communication_ports {}
}