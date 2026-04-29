/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

# Cloud resource owner - for cost tracking
variable "Owner" {
  type        = string
  description = "Resource Owner"
}

variable "Region" {
  type        = string
  description = "Region to be used to configure provider"
}

variable "StageName" {
  type        = string
  description = "Control Plane stage name"
}

variable "SaxCell" {
  type        = string
  description = "The SAX Cell being used for configuration of OIDC Role"
}

variable "Fast" {
  type        = string
  description = "The ID of the default Fast LLM Model"
}

variable "Smart" {
  type        = string
  description = "The ID of the default Smart LLM Model"
}

variable "Pro" {
  type        = string
  description = "The ID of the default Pro LLM Model"
}

variable "AccountID" {
  type        = string
  description = "GenAI Account ID where LLM Models are deployed"
}

# Variables below are populated by Provisioning Services
# Do not rename these variables

variable "RoleArn" {
  type        = string
  description = "Provided by Provisioning Services for target Cluster account"
}

variable "provisioningTags" {
  type        = map(any)
  default     = {}
  description = "Populated by Provisioning Services with Pega standard tags"
}