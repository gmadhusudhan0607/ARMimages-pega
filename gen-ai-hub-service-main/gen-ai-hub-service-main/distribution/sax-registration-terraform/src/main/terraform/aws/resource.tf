/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

// Service
resource "pegasec_backing_service" "GenAIGatewayService" {
  name     = "genai-gateway-service"
  audience = "backing-services"
  scopes = [
    "read",
    "write",
    "swagger"
  ]
}

data "pegasec_backing_service" "GenAIGatewayService" {
  depends_on = [pegasec_backing_service.GenAIGatewayService]
  name       = "genai-gateway-service"
  audience   = "backing-services"
  region     = var.Region
  // Region parameter helps choose the closest geographically Okta deployment.
  // Based on region it can be either US, EMEA or APAC Okta deployment - latency reasons.
}