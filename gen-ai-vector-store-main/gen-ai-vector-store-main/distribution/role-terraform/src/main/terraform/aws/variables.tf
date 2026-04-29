/*
 * Copyright (c) 2023 Pegasystems Inc.
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

variable "ActiveRegion" {
  type        = string
  description = "Active AWS region to use (MRDR)"
}

variable "ResourceGUID" {
  type        = string
  description = "CMDB resource GUID"
}

variable "DatabaseSecret" {
  type    = string
  default = ""
}

variable "ActiveDatabaseSecret" {
  type    = string
  default = ""
}

variable "SaxClientSecret" {
  type    = string
  default = ""
}

variable "KMSKeyForSecrets" {
  type    = string
  default = ""
}

variable "ActiveKMSKeyForSecrets" {
  type    = string
  description = "KMSKeyForSecrets in Active Region (MRDR)"
  default = ""
}

variable ClusterOIDCIssuerURL {
  type = string
}

variable "RoleArn" {
  type    = string
  default = ""
}

variable "DatabaseID" {
  type    = string
  default = ""
}

variable "provisioningTags" {
  type    = map(any)
  default = {}
}
