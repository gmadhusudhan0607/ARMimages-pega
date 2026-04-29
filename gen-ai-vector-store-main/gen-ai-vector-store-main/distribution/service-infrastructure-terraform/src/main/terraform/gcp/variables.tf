/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

# Parameter specifies who's the cloud resource owner - cost tracking
variable "Owner" {
  type    = string
}

variable "Region" {
  type        = string
  description = "GCP region to use"
}

# Parameter populated by PS
variable "provisioningTags" {
  type    = map(any)
  default = {}
}

# Parameter populated by PS
variable "cloud_provider" {
  default = "gcp"
  type    = string
}

# Parameter populated by PS
variable "RoleArn" {
  type    = string
  default = ""
}

variable AccountID {
  type        = string
  default     = ""
  description = "GCP Project ID"
}

