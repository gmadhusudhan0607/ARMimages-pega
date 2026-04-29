//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database configuration access in readonly mode", func() {

	_ = Context("when retrieving configuration", func() {

		It("should return database configuration successfully", func() {
			By("Requesting database configuration")
			uri := fmt.Sprintf("%s/v1/db/configuration", baseOpsURL)

			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response []map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response).NotTo(BeNil())
			// DB configuration should be readable in readonly mode
		})
	})
})
