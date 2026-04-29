/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
# Parameter specifies who's the cloud resource owner - cost tracking
variable "Owner" {
  type        = string
  description = "Resource owner for labelling purpose"
}

variable "Region" {
  type        = string
  description = "Region to be used to configure provider"
}

variable "InferenceRegion" {
  type        = string
  default     = ""
  description = "AWS Region for the model endpoint to be invoked."
}

variable "ModelMapping" {
  type        = string
  description = "Which endpoint of the GenAI Gateway must provide this Model"
}

variable "ModelID" {
  type        = string
  description = "The ID of the AWS Bedrock Model"
}

variable "TargetApi" {
  type        = string
  description = "The Bedrock API to be used for inference calls"
}

variable "UseRegionalInferenceProfile" {
  type        = bool
  description = "Use regional inference profile for this model calls"
}

variable "SaxCell" {
  type        = string
  description = "The SAX Cell being used for configuration of OIDC Role"
}

variable "OidcProviderUrl" {
  type        = string
  description = "The OIDC Provider Endpoint"
}

variable "Inactive" {
  type        = bool
  default     = false
  description = "When true, the Gateway ignores this model mapping and Terraform skips Bedrock API data source lookups, allowing plan/apply/destroy without querying AWS for model existence."
}

variable "AccountID" {
  type        = string
  description = "GenAI Account ID to provision Bedrock resources."
}

variable "StageName" {
  type        = string
  description = "Control Plane stage name"
}

variable "InferenceProfilePrefix" {
  type        = string
  default     = ""
  description = "CRIS prefix resolved conditionally - only populated when UseRegionalInferenceProfile is true"
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
  description = "Populated by Provisioning Services with Pega standard tags (do not rename this variable)."
}
