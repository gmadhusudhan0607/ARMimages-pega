/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

locals {
  tags = merge(
    var.provisioningTags,
    tomap({
      "Owner" = var.Owner
    })
  )
}