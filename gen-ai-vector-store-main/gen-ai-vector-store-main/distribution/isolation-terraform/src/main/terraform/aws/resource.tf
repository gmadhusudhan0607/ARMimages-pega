/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

locals {
  path = "/v1/isolations"
  pathRO = "/v1/isolationsRO"
  debug  = true
  effective_path = var.DeploymentMode == "active" ? local.path : local.pathRO
}

# Create new isolation
resource "ps-restapi_object" "post_isolation" {
  debug = local.debug
  path = local.effective_path
  object_id = var.Isolation
  data = "{ \"id\": \"${var.Isolation}\", \"maxStorageSize\": \"${var.MaxStorageSize}\", \"pdcEndpointURL\": \"${var.PDCEndpointURL}\" }"
  timeout = "5m"
  extract_fields = ["id"]
}

# Get newly created isolation to verify it has been created.
data "ps-restapi_object" "get_isolation" {
  debug = local.debug
  path = local.effective_path
  object_id = var.Isolation

  depends_on = [ps-restapi_object.post_isolation]
  extract_fields = [
    "id",
    "maxStorageSize",
    "pdcEndpointURL"
  ]
}

