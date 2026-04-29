/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// Infrastructure represents the deployment infrastructure
type Infrastructure string

const (
	InfrastructureAWS   Infrastructure = "aws"
	InfrastructureGCP   Infrastructure = "gcp"
	InfrastructureAzure Infrastructure = "azure"
)
