/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

// Service
resource "pegasec_backing_service" "GenAIVectorStore" {
  name     = "genai-vector-store"
  audience = "backing-services"
  scopes = [
    "read",
    "write",
    "swagger"
  ]
}

data "pegasec_backing_service" "GenAIVectorStore" {
  depends_on = [pegasec_backing_service.GenAIVectorStore]
  name       = "genai-vector-store"
  audience   = "backing-services"
  region     = var.Region
  // Region parameter helps choose the closest geographically Okta deployment.
  // Based on region it can be either US, EMEA or APAC Okta deployment - latency reasons.
}

####################################################################################################
// Ops Service
resource "pegasec_backing_service" "GenAIVectorStoreOps" {
  name     = "genai-vector-store-ops"
  audience = "cp-services"
  scopes   = [
    "isolations.read",
    "isolations.write",
    "operations.read",
    "operations.write",
    "swagger"
  ]
}

data "pegasec_backing_service" "GenAIVectorStoreOps" {
  depends_on = [pegasec_backing_service.GenAIVectorStoreOps]
  name       = "genai-vector-store-ops"
  audience   = "cp-services"
  region     = var.Region == "us-gov-west-1"? var.Region : "us-east-1"
  // in this case for EDR support, the implementation above is correct as regions are mapped to US Okta cell
  // for more information please see https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-710966
  // In the future we may be able to use cp-services in any region.
  // For more information see https://knowledgehub.pega.com/SERVAUTH:How_to_integrate_and_use_SAX#What_service_type_should_I_use?
}
