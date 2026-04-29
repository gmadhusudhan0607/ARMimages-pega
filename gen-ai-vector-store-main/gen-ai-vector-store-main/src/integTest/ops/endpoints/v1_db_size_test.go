// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.
package endpoints_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing OPS /v1/db/size", func() {

	_ = Context("when retrieving database size", func() {

		It("should return database size successfully", func() {
			By("Requesting database size")
			uri := fmt.Sprintf("%s/v1/db/size", baseOpsURI)

			resp, body, err := HttpCall("GET", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response).To(HaveKey("used_bytes"))
			Expect(response["used_bytes"]).To(BeNumerically(">", 0))
		})
	})
})
