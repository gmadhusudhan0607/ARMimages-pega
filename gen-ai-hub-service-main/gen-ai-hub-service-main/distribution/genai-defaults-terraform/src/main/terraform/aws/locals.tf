/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


locals {

  genai_defaults_secret_json = jsonencode({
    "fast"   = var.Fast
    "smart"  = var.Smart
    "pro"    = var.Pro
  })

  genai_defaults_secret_name = "genai_infra/defaults/${var.StageName}/${var.SaxCell}/defaults"

  tags = merge(
    var.provisioningTags,
    tomap({
      "Owner" = var.Owner
    })
  )
}