/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.58.0"
    }
    pegasec = {
      source  = "pega.com/cloud/pegasec"
      version = "~> 1.4" //"${var.pegasec_version}"
    }
  }
  required_version = ">= 1.0"
}
