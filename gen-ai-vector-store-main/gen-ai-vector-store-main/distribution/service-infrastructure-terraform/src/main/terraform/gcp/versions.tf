/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~>5.0.0"
    }
    pegasec = {
      source  = "pega.com/cloud/pegasec"
      version = "1.2.11" //"${var.pegasec_version}"
    }
  }
  required_version = ">= 1.0"
}
