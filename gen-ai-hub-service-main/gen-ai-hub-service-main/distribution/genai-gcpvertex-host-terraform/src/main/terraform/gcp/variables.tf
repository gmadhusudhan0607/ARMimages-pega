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

variable "provisioningTags" {
  type        = map(any)
  default     = {}
  description = "Populated by Provisioning Services with Pega standard tags (do not rename this variable)."
}


