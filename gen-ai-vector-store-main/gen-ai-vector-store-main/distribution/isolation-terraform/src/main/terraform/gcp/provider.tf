/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

provider "ps-restapi" {
  uri    = var.OpsServiceEndpoint
  debug  = local.debug
  sax    = true
  scopes = [
    "pega.genai-vector-store-ops:isolations.read",
    "pega.genai-vector-store-ops:isolations.write"
  ]
}