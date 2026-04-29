/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

variable "AccountID" {
  type        = string
  description = "GCP Project ID"
}

variable "Region" {
  type        = string
  description = "GCP region to use"
}

variable "ActiveRegion" {
  type        = string
  description = "Active GCP region to use (EDR)"
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

variable "RoleArn" {
  type    = string
  default = ""
}

variable "Namespace" {
  type = string
}

variable "Owner" {
  type    = string
  default = ""
}

variable "provisioningTags" {
  type = map(any)
  default = {}
}

variable "DatabaseID" {
  type    = string
  default = ""
}

variable "SaxClientSecret" {
  type    = string
  default = ""
}
