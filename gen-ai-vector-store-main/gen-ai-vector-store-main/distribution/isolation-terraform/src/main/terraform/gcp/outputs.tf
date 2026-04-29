/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

# Return data we received from Ops service as confirmation that isolation exist in DB
output "IsolationID" {
  value = try(nonsensitive(data.ps-restapi_object.get_isolation.api_data.id), "nil")
}

output "MaxStorageSize" {
  value = try(nonsensitive(data.ps-restapi_object.get_isolation.api_data.maxStorageSize), "nil")
}

output "VSPDCEndpointURL" {
  value = try(nonsensitive(data.ps-restapi_object.get_isolation.api_data.pdcEndpointURL), "nil")
}
