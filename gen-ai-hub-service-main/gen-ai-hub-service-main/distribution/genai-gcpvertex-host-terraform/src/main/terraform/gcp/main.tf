/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

# Generate a random suffix for resource names
resource "random_id" "suffix" {
  byte_length = 4
}

# Deploy placeholder Cloud Run service to get the URL for API spec host
# This solves the chicken-and-egg problem where we need the ESPv2 Cloud Run URL
# before we can create the Endpoints service, but we need the Endpoints service
# before we can deploy ESPv2
resource "null_resource" "espv2_placeholder" {
  provisioner "local-exec" {
    command = <<-EOT
      gcloud run deploy ${local.endpoints_service_name} \
        --image="gcr.io/cloudrun/hello" \
        --allow-unauthenticated \
        --platform managed \
        --region=${var.Region} \
        --project=${var.GcpProjectId}
    EOT
  }

  # Trigger re-deployment if service name changes
  triggers = {
    service_name = local.endpoints_service_name
    project_id   = var.GcpProjectId
    region       = var.Region
  }
}

data "external" "placeholder_url" {
  program = ["bash", "-c", <<-EOF
    # Check if service exists to avoid errors during destroy
    if gcloud run services describe ${local.endpoints_service_name} --region=${var.Region} --project=${var.GcpProjectId} --quiet >/dev/null 2>&1; then
      URL=$(gcloud run services describe ${local.endpoints_service_name} --region=${var.Region} --project=${var.GcpProjectId} --format="value(status.url)")
      echo "{\"url\":\"$URL\"}"
    else
      echo "{\"url\":\"\"}"
    fi
EOF
  ]

  depends_on = [null_resource.espv2_placeholder]
}
