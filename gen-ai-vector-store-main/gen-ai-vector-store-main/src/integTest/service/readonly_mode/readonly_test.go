//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"fmt"
	"net/url"
)

// Helper functions used by multiple test files

func getDocumentsGetEndpoint(endpointURI, docID string) string {
	u, err := url.Parse(endpointURI)
	if err != nil {
		panic(err)
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, docID)
	return u.String()
}

func getDocumentsPatchEndpoint(endpointURI, docID string) string {
	u, err := url.Parse(endpointURI)
	if err != nil {
		panic(err)
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, docID)
	return u.String()
}

func getDocumentsDeleteByIdEndpoint(endpointURI, docID string) string {
	u, err := url.Parse(endpointURI)
	if err != nil {
		panic(err)
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, docID)
	return u.String()
}
