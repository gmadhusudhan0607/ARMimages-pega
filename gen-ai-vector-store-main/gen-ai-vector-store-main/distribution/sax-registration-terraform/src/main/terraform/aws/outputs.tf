/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################


# Storage Service
########################################################################################
output "SaxJWKSEndpoint" {
  value = data.pegasec_backing_service.GenAIVectorStore.jwks_endpoint
}

output "SaxIssuer" {
  value = data.pegasec_backing_service.GenAIVectorStore.issuer
}

output "SaxAudience" {
  value = data.pegasec_backing_service.GenAIVectorStore.audience
}

output "SaxScopesString" {
  value = data.pegasec_backing_service.GenAIVectorStore.scopes_string
}


# Storage Ops Service
########################################################################################
output "SaxOpsJWKSEndpoint" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.jwks_endpoint
}

output "SaxOpsIssuer" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.issuer
}

output "SaxOpsAudience" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.audience
}

output "SaxOpsScopesString" {
  value = data.pegasec_backing_service.GenAIVectorStoreOps.scopes_string
}
