//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"fmt"
	"net/http"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC /v1/*/smart-attributes-group in Emulation mode", func() {

	var isolationID string
	var endpointURI string

	_ = Context("service is ready", func() {

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			endpointURI = fmt.Sprintf("%s/v1/%s/smart-attributes-group", svcBaseURI, isolationID)
		})

		It("list attribute groups successfully", func() {

			By("Calling GET attributes group endpoint")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validate response schema matches service.yaml specification")
			attrGroupsList, err := ValidateAttributesGroupsListResponse(body)
			Expect(err).To(BeNil())
			Expect(attrGroupsList).NotTo(BeNil())

		})
		It("GET attribute group by ID", func() {

			By("Calling GET attributes group endpoint for specific ID")
			uri := fmt.Sprintf("%s/%s", endpointURI, "test-group-1")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validate response schema matches service.yaml specification")
			attrGroup, err := ValidateAttributesGroupResponse(body)
			Expect(err).To(BeNil())
			Expect(attrGroup).NotTo(BeNil())
			Expect(attrGroup.GroupID).NotTo(BeEmpty())

		})

		It("POST smart attributes group", func() {

			By("Attempting to create attributes group in EmulationMode")
			description := fmt.Sprintf("Attributes group for test %s ", isolationID)
			reqBody := fmt.Sprintf(`{"description":"%s","attributes":["version","category"]}`, description)
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Validate response schema matches service.yaml specification")
			attrGroupResp, err := ValidateAttributesGroupCreationResponse(body)
			Expect(err).To(BeNil())
			Expect(attrGroupResp).NotTo(BeNil())
			Expect(attrGroupResp.GroupID).NotTo(BeEmpty())
			Expect(attrGroupResp.Description).NotTo(BeEmpty())
			Expect(attrGroupResp.Attributes).NotTo(BeNil())

		})
		It("DELETE smart attributes group", func() {

			By("Attempting to delete attributes group in EmulationMode")
			uri := fmt.Sprintf("%s/%s", endpointURI, "test-group-1")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("DELETE", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Validate response schema matches service.yaml specification")
			err = ValidateEmptyResponse(body)
			Expect(err).To(BeNil())
		})
	})
})

type ListAttributesGroupsResp struct {
	GroupID     string `json:"groupID" binding:"required"`
	Description string `json:"description,omitempty"`
}
