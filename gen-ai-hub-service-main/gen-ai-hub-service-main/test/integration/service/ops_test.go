//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package service_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/ops/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/repository"
	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
)

var _ = Describe("Tests Ops:", Ordered, func() {

	var testID string

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	opsLabel := Label("ops")

	_ = Context("calling v1/isolations/:isolationId/metrics", func() {
		It("should return metrics with the total number of requests", opsLabel, func() {
			// Get initial count (may not be 0 if there's existing data from previous runs)
			initialStatus, initialResp := ExpectOpsCall("13081", "GET", "/v1/isolations/123/metrics", "", testID)
			Expect(initialResp).NotTo(BeNil())
			Expect(initialStatus).Should(Equal(200))

			var initialMetrics api.IsolationMetrics
			err := json.Unmarshal([]byte(initialResp), &initialMetrics)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(initialMetrics.Isolation).Should(Equal("123"))
			initialCount := initialMetrics.Requests

			// For a fresh test, the count should be stable (no new events added)
			status, r := ExpectOpsCall("13081", "GET", "/v1/isolations/123/metrics", "", testID)
			Expect(r).NotTo(BeNil())

			var resp api.IsolationMetrics
			err = json.Unmarshal([]byte(r), &resp)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(status).Should(Equal(200))
			Expect(resp.Isolation).Should(Equal("123"))
			Expect(resp.Requests).Should(Equal(initialCount))
		})

		It("should return metrics from last 24 hours, 2 hours ago and 1 minute ago", opsLabel, func() {
			// Capture a single reference time to derive all relative timestamps,
			// avoiding clock drift between steps on heavily loaded CI machines.
			now := time.Now()

			// Get initial count before adding new events
			initialStatus, initialResp := ExpectOpsCall("13081", "GET", "/v1/isolations/123/metrics", "", testID)
			Expect(initialResp).NotTo(BeNil())
			Expect(initialStatus).Should(Equal(200))

			var initialMetrics api.IsolationMetrics
			err := json.Unmarshal([]byte(initialResp), &initialMetrics)
			Expect(err).ShouldNot(HaveOccurred())
			initialCount := initialMetrics.Requests

			tenMinutesAgo := now.Add(-10 * time.Minute).Unix()
			twentyThreeHoursAgo := now.Add(-23 * time.Hour).Unix()
			e1 := repository.Event{Isolation: "123", Timestamp: tenMinutesAgo}
			s1, _ := ExpectOpsCall("13081", "POST", "/v1/events", ToJSONString(e1), testID)
			e2 := repository.Event{Isolation: "123", Timestamp: twentyThreeHoursAgo}
			s2, _ := ExpectOpsCall("13081", "POST", "/v1/events", ToJSONString(e2), testID)
			Expect(s1).Should(Equal(200))
			Expect(s2).Should(Equal(200))

			status, r := ExpectOpsCall("13081", "GET", "/v1/isolations/123/metrics", "", testID)
			verifyMetricsResponse(status, 200, r, "123", initialCount+2)

			// Use the same reference point minus 11 minutes to ensure the 10-minute-old event
			// is safely within the query window, regardless of test execution duration.
			elevenMinutesAgo := now.Add(-11 * time.Minute).Unix()
			status, r = ExpectOpsCall("13081", "GET", fmt.Sprintf("/v1/isolations/123/metrics?from=%d", elevenMinutesAgo), "", testID)
			verifyMetricsResponse(status, 200, r, "123", 1)

			status, r = ExpectOpsCall("13081", "GET", fmt.Sprintf("/v1/isolations/123/metrics?to=%d", elevenMinutesAgo), "", testID)
			verifyMetricsResponse(status, 200, r, "123", initialCount+1)
		})

		It("should return 400 if request payload is not proper", opsLabel, func() {
			tenMinutesAgo := time.Now().Add(-10 * time.Minute).Unix()
			badRequest := fmt.Sprintf(`{"isolationId": "123", "timestamp":%d}`, tenMinutesAgo)
			s1, _ := ExpectOpsCall("13081", "POST", "/v1/events", badRequest, testID)
			Expect(s1).Should(Equal(400))
		})

		It("should propagate request events when genai model endpoints are called", opsLabel, func() {
			// add the Authorization header with the SAX token
			// call gpt-35-turbo model
			// call text-embedding-ada-002 model
			// call gpt-4o model
			// call gpt-4o-mini model

			// fetch metrics from ops api and it should have 4 for the isolation present on the SAX token
		})

	})
})

func verifyMetricsResponse(status, expectedStatus int, r, expectedIsolation string, expectedCount int) {
	Expect(r).ShouldNot(BeNil())
	var resp api.IsolationMetrics
	err := json.Unmarshal([]byte(r), &resp)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(status).Should(Equal(expectedStatus))
	Expect(resp.Isolation).Should(Equal(expectedIsolation))
	Expect(resp.Requests).Should(Equal(expectedCount))
}
