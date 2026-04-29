/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

# Parameter specifies who's the cloud resource owner - cost tracking
variable "Owner" {
  type        = string
  description = "Resource owner for labelling purpose"
}

variable "AccountID" {
  type        = string
  description = "GenAI Account to configure the OIDC Provider"
}

variable "Region" {
  type        = string
  description = "Reference Region for the Control Plane deployment"
}

variable "SaxCell" {
  type        = string
  description = "SAX stage being used for OIDC configuration"
}

variable "StageName" {
  type        = string
  description = "Control Plane stage name"
}

# Variables below are populated by Provisioning Services
# Do not rename these variables
variable "provisioningTags" {
  type        = map(any)
  default     = {}
  description = "Populated by Provisioning Services with Pega standard tags."
}

