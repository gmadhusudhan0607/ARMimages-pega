/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

locals {
  //Tags in GCP have to be in lower case, and with limited special characters
  ps_tags = merge(
    var.provisioningTags,
    tomap({
      owner = var.Owner
    })
  )
  tags = {
    for k, v in local.ps_tags :
    replace(lower(k), "/[,./:]/", "_") => replace(lower(v), "/[,./:]/", "_")
  }
  //Using toset to remove duplicates and guarantee that the list of ARN is unique (in case both values point to the same secret)
  DBSecretsARNs = toset([var.DatabaseSecret, var.ActiveDatabaseSecret])

}