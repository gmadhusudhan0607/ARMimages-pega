/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

data "pegasec_backing_service" "GenAIVectorStore" {
  name       = "genai-vector-store"
  audience   = "backing-services"
  region     = var.Region
  // Region parameter helps choose the closest geographically Okta deployment.
  // Based on region it can be either US, EMEA or APAC Okta deployment - latency reasons.
}


data "pegasec_backing_service" "GenAIVectorStoreOps" {
  name       = "genai-vector-store-ops"
  audience   = "cp-services"
  region     = var.Region == "us-gov-west-1"? var.Region : "us-east-1"
  // "cp-services" only supports "us-east-1" region by default. A support was added to "us-gov-west-1".
  // In the future we may be able to use cp-services in any region.
  // For more information see https://knowledgehub.pega.com/SERVAUTH:How_to_integrate_and_use_SAX#What_service_type_should_I_use?
}

