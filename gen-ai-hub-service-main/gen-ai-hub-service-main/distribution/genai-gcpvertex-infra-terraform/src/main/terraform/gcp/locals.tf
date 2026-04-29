/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

locals {

  suffix = var.ResourcesSuffixId

  cloud_run_name       = "genai-vertex-gemini-function-${var.ResourcesSuffixId}"
  bucket_name          = "genai-vertex-function-source-${var.ResourcesSuffixId}"
  service_account_name = "genai-vertex-invoker-${var.ResourcesSuffixId}"
  apigw_sa_name        = "genai-apigw-sa-${var.ResourcesSuffixId}"

  endpoints_service_name = var.GcpVertexAIServiceName
  endpoints_host         = var.GcpVertexAIApiGatewayHost

  google_endpoint_service_name = "${local.endpoints_service_name}.endpoints.${var.GcpProjectId}.cloud.goog"

  function_timeout_seconds = tonumber(var.FunctionTimeoutSeconds)

  spec_file = templatefile("${path.module}/templates/api-v3-spec.yaml.tpl", {
    oidc_issuer              = var.OidcIssuer
    function_url             = module.vertex_function.function_url
    apihost                  = local.endpoints_host
    suffix                   = local.suffix
    function_timeout_seconds = local.function_timeout_seconds
  })

  provisioning_tags = merge(
    var.provisioningTags,
    {
      owner = var.Owner
    }
  )
}
