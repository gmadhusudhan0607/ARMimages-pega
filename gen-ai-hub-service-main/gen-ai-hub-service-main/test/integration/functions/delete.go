//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func DeleteMockServerExpectation(mockServerURL, expectationID string) {
	By(fmt.Sprintf("-> Deleting mockserver expectation %s", expectationID))
	uri := fmt.Sprintf("%s/mockserver/clear", mockServerURL)
	jsonData := fmt.Sprintf("{ \"id\": \"%s\" }", expectationID)

	resp, _, err := ExpectHttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func DeleteModelExpectation(mockServerURL string, model Model) {
	if model.Expectation != nil && model.Expectation.Id != "" {
		DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
	}
}

func DeleteBuddyExpectation(mockServerURL string, buddy Buddy) {
	if buddy.Expectation != nil && buddy.Expectation.Id != "" {
		DeleteMockServerExpectation(mockServerURL, buddy.Expectation.Id)
	}
}
