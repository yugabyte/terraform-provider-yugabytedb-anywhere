data "yb_storage_configs" "configs" {
    // To fetch any storage config
}

data "yb_storage_configs" "configs_gcs" {
    // To fetch id of a particular stoage config
    config_name = "<storage-config-name>"
}
