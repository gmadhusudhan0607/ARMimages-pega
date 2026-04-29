/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

# Parameter specifies who's the cloud resource owner - cost tracking
variable "Owner" {
  type    = string
}

# Parameter required by SAX
variable "Region" {
  type    = string
  default = "us-east-1"
}

# Parameter populated by PS
variable "provisioningTags" {
  type    = map(any)
  default = {}
}

# Parameter populated by PS
variable "cloud_provider" {
  default = "aws"
  type    = string
}

# Parameter populated by PS
variable "RoleArn" {
  type    = string
  default = ""
}

# Parameter populated by PS
variable "pe_env_variables_aws_auth" {
  type    = map(any)
  default = { AWS_PROFILE = "eks" }
}
