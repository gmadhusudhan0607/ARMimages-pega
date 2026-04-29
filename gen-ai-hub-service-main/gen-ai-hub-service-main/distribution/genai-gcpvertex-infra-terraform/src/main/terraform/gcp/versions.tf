/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "7.12.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "7.12.0"
    }


    http = {
      source  = "hashicorp/http"
      version = "~> 3.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
  required_version = ">= 1.0"
}