/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

variable "AccountID" {
  type        = string
  description = "AWS Account ID"
}

variable "Region" {
  type        = string
  description = "AWS region to use"
}

variable "ResourceGUID" {
  type        = string
  description = "CMDB resource GUID"
}

variable "SaxClientSecret" {
  type    = string
  default = ""
}

variable "KMSKeyForSecrets" {
  type    = string
  default = ""
}

variable ClusterOIDCIssuerURL {
  type = string
}

variable "RoleArn" {
  type    = string
  default = ""
}

variable "provisioningTags" {
  type    = map(any)
  default = {}
}