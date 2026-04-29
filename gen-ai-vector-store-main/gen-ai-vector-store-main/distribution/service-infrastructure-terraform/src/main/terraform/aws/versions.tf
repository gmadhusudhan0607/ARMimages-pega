/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.83.1" #for >= 5.83.1 but < 5.84.0
    }
    pegasec = {
      source  = "pega.com/cloud/pegasec"
      version = "1.2.11" //"${var.pegasec_version}"
    }
  }
  required_version = ">= 1.0"
}
