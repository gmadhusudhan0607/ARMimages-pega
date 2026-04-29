/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

locals {
  # set the role to be assumed by the Provider to configure resource in target account
  stage_name         = var.StageName == "production-usgov" ? "prod-usgov" : var.StageName
  llmAccountRoleArn  = "arn:${data.aws_partition.cp_partition.partition}:iam::${var.AccountID}:role/provisioning-service-${local.stage_name}-PEAccessRole"

  backingServicesAud = "backing-services"
  tags = merge(
    var.provisioningTags,
    tomap({
      "Owner" = var.Owner
    })
  )

  # https://knowledgehub.pega.com/SERVAUTH:How_to_integrate_and_use_SAX#Appendix_2._GUIDs_for_backing_services_registration
  sax_stage_map = {
    integration    = "staging"
    staging        = "staging"
    trials         = "staging"
    prod-adoption  = "production"
    production     = "production"
    prod-launchpad = "prod-launchpad"
    rnd-usgov      = "rnd-usgov"
    production-usgov  = "production-usgov"
  }

  sax_stage = local.sax_stage_map[var.StageName]

  sax_endpoints = {
    staging = {
      us = "https://stg-fcp-us-1.oktapreview.com/oauth2/ausg5ldmi6IpvdpXX1d6"
      eu = "https://stg-fcp-emea-1.oktapreview.com/oauth2/aus1b6ia6ppUvChzM0x7"
    }
    production = {
      us   = "https://fcp-us-1.okta.com/oauth2/aus9q9t92JHq6oZEC5d6"
      eu   = "https://fcp-emea-1.okta.com/oauth2/ausgi9g1w89Lm0MHf416"
      apac = "https://fcp-apac-1.okta.com/oauth2/auso0fj71B4lKRG873l6"
    }
    prod-launchpad = {
      us = "https://lpcp-us-1.okta.com/oauth2/aus6kraa58BpIyMMv697"
      eu = "https://lpcp-emea-1.okta.com/oauth2/aus8e02v6rrdkiTYZ417"
    }
    rnd-usgov = {
      us = "https://gov-fcp-us.oktapreview.com/oauth2/ausi4sbyub6241HZv1d7"
    }
    production-usgov = {
      us = "https://gov-fcp-us.okta-gov.com/oauth2/ausxlcv84DCEmHnJD0j6"
    }
  }

  sax_issuer_url = local.sax_endpoints[local.sax_stage][var.SaxCell]
}