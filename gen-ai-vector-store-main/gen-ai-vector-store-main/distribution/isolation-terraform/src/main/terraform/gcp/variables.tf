/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

variable "OpsServiceEndpoint" {
  type        = string
  description = "GenAI Vector Store Ops Service URL"
}

variable "Isolation" { # used name 'Isolation' instead if 'IsolationID' to avoid conflict with SCE dynamic parameter name
  type        = string
  description = "Isolation ID"
}

variable "MaxStorageSize" {
  type        = string
  description = "Max Storage Size"
}

variable "PDCEndpointURL" {
  type        = string
  description = "PDC Endpoint URL"
  default     = ""
}

variable "DeploymentMode" {
  type        = string
  description = "Deployment mode, should be 'active' to allow deletion of isolation"
}

#####################

variable "AccountID" {
  type        = string
  description = "AWS Account ID"
}

variable "Region" {
  type        = string
  description = "AWS region to use"
}

variable "RoleArn" {
  type    = string
  default = ""
}

variable "provisioningTags" {
  type    = map(any)
  default = {}
}
