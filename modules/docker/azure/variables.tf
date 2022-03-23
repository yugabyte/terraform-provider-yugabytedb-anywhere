variable "cluster_name" {
  description = "Name for YugabyteDB Anywhere cluster"
  type        = string
  default     = "yugaware"
}
variable "region_name" {
  description = "region to use for resources"
  type        = string
}
variable "vm_size" {
  description = "vm specs"
  type        = string
  default     = "Standard_D4s_v3"
}
variable "disk_size" {
  description = "disk size"
  type        = string
  default     = "100"
}
variable "ssh_user" {
  description = "name of the ssh user"
  type        = string
}
variable "subnet_name" {
  description = "name of the subnet to use for the YugabyteDB Anywhere instance"
  type        = string
}
variable "vnet_name" {
  description = "name of the virtual network to use for the YugabyteDB Anywhere instance"
  type        = string
}
variable "vnet_resource_group" {
  description = "name of the resource group associated with the virtual network"
  type        = string
}

// files
variable "ssh_private_key" {
  description = "Path to private key to use when connecting to the instances"
  type        = string
}
variable "ssh_public_key" {
  description = "Path to SSH public key to be use when creating the instances"
  type        = string
}