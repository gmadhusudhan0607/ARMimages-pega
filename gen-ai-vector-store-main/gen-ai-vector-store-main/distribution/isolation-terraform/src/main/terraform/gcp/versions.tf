/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

terraform {
  required_providers {
    ps-restapi = {
      source  = "pega.com/cloud/ps-restapi"
      version = "~>0.1.21" # Note: Actual version is injected by gradle, this is used to be able to run terraform locally
    }
  }
  required_version = ">= 1.0"
}
