/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

locals {
  suffix               = random_id.suffix.hex

  endpoints_service_name = "genai-vertex-gateway-${local.suffix}"

  # Extract the host from the placeholder Cloud Run service URL
  endpoints_host_url = data.external.placeholder_url.result.url
  endpoints_host = replace(local.endpoints_host_url, "https://", "")
  
  provisioning_tags = merge(
    var.provisioningTags,
    {
      owner = var.Owner
    }
  )
}
