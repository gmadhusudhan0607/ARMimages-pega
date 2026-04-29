/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

variable "ClusterGUID" {
  type        = string
  description = "GUID of CloudK cluster where service is deployed"
}

variable "KmsKeyId" {
  type        = string
  description = "Encryption Key ID used to encrypt the Secret"
}

variable "Model" {
  type        = string
  description = "The kind of model that is being configured"
}

variable "ModelProvider" {
  type        = string
  description = "The provider for the model"
}

variable "VersionCurrent" {
  type        = string
  description = "The general supported model version (Ex: 1106, 002, 20240210)"
  default     = ""
}

variable "VersionDeprecated" {
  type        = string
  description = "The model version being deprecated, end-of-life or being replaced (Ex: 1106, 002, 20240210)"
  default     = ""
}

variable "VersionNext" {
  type        = string
  description = "The model version being rolled out as new general available version supported (Ex: 1106, 002, 20240210)"
  default     = ""
}

variable "ModelEndpoint" {
  type        = string
  description = "The URL endpoint that is accessible to GenAI Gateway Service for model inference calls"
  default     = ""
}

variable "APIKey" {
  type        = string
  description = "The API Key that grants access to the Generative Model Endpoint"
  default     = ""
}

# Parameter specifies who's the cloud resource owner - cost tracking
variable "Owner" {
  type = string
  description = "Resource owner for labelling purpose"
}

variable "Active" {
  type = string
  description = "The Model mapping is active"
  default = "false"
}

variable "Region" {
  type = string
  description = "Region to be used to configure provider"
}

variable "provisioningTags" {
  type    = map(any)
  default = {}
  description = "Populated by Provisioning Services with Pega standard tags"
}

variable "RoleArn" {
  type = string
  description = "Populated by Provisioning Services with Role to be assumed to configure provider"
}