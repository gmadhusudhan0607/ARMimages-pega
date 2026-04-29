/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################


output "ServiceAccountRole" {
  value = google_service_account.vector_store_service_account.email
}
