// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"regexp"
	"strings"
)

// ConvertUsageDataURL converts the original usage data URL to the correct format for usage data upload
// Based on the reference implementation:
// 1. Replace 'PRSOAPServlet' with 'PRRestService'
// 2. Use regex to insert '/PegaUVU/v1/UsageDataFile' after the UUID part and remove any trailing path
func ConvertUsageDataURL(originalURL string) string {
	// Replace 'PRSOAPServlet' with 'PRRestService'
	updatedURL := strings.Replace(originalURL, "PRSOAPServlet", "PRRestService", -1)

	// Use regex to insert '/PegaUVU/v1/UsageDataFile' after the UUID part and remove any trailing path
	re := regexp.MustCompile(`(/PRRestService/[^/]+).*`)
	updatedURL = re.ReplaceAllString(updatedURL, `$1/PegaUVU/v1/UsageDataFile`)

	return updatedURL
}
