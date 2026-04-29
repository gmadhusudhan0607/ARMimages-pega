/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

## Service
output "SaxIssuer" {
  value = data.pegasec_backing_service.GenAIVectorStore.issuer
}

output "SaxJWKSEndpoint" {
  value = data.pegasec_backing_service.GenAIVectorStore.jwks_endpoint
}

output "SaxAudience" {
  value = data.pegasec_backing_service.GenAIVectorStore.audience
}


## Ops Service
output "SaxOpsIssuer" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.issuer
}

output "SaxOpsJWKSEndpoint" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.jwks_endpoint
}

output "SaxOpsAudience" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.audience
}
