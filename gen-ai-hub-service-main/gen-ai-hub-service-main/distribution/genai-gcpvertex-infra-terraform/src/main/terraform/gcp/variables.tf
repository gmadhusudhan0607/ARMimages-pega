/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

variable "Owner" {
  type        = string
  description = "Resource owner for labelling purpose"
}

variable "GcpProjectId" {
  description = "The GCP project ID"
  type        = string
}

variable "Region" {
  description = "The GCP region"
  type        = string
}

variable "ResourcesSuffixId" {
  type = string
}

variable "GcpVertexAIServiceName" {
  type = string
}

variable "GcpVertexAIApiGatewayHost" {
  type = string
}

variable "OidcIssuer" {
  description = "OIDC issuer for API Gateway security"
  type        = string
}

variable "provisioningTags" {
  type        = map(any)
  default     = {}
  description = "Populated by Provisioning Services with Pega standard tags (do not rename this variable)."
}

variable "ModelInferenceRegionOverride" {
  description = "The specific region to use for Vertex AI inference. If provided, overrides the default region detection."
  type        = string
  default     = null
}

variable "FunctionTimeoutSeconds" {
  description = "Timeout in seconds for the Cloud Function execution and client calls"
  type        = string
  default     = "600"
}
