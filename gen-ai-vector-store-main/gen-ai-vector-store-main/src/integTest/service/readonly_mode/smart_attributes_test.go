//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	attributesgroup "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes_group"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC /v1/*/smart-attributes-group in ReadOnly mode", func() {

	ctx := context.Background()
	var isolationID string
	var endpointURI string
	var endpointURISetup string

	_ = Context("service is ready", func() {

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			endpointURI = fmt.Sprintf("%s/v1/%s/smart-attributes-group", svcBaseURI, isolationID)
			endpointURISetup = fmt.Sprintf("%s/v1/%s/smart-attributes-group", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			// Create attribute groups for testing
			CreateAttributeGroup(ctx, database, endpointURISetup, isolationID, "x", []string{RandStringRunes(5)})
			CreateAttributeGroup(ctx, database, endpointURISetup, isolationID, "y", []string{RandStringRunes(5)})
			CreateAttributeGroup(ctx, database, endpointURISetup, isolationID, "z", []string{RandStringRunes(5)})
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
		})

		It("list attribute groups successfully", func() {

			By(fmt.Sprintf("Calling GET attributes group endpoint"))
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By(fmt.Sprintf("Validate list attributes groups endpoint returns all created groups"))
			var attrGroups []ListAttributesGroupsResp
			err = json.Unmarshal(body, &attrGroups)
			Expect(err).To(BeNil())
			Expect(len(attrGroups)).To(Equal(3))

		})
		It("GET attribute group by ID", func() {

			By(fmt.Sprintf("Creating attributes group"))
			rndAttributes := []string{RandStringRunes(5), RandStringRunes(5), RandStringRunes(5)}
			description := fmt.Sprintf("Attributes group for test %s ", isolationID)
			attrGroupID := CreateAttributeGroup(ctx, database, endpointURISetup, isolationID, description, rndAttributes)

			By(fmt.Sprintf("Calling GET attributes group endpoint "))
			uri := fmt.Sprintf("%s/%s", endpointURI, attrGroupID)
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By(fmt.Sprintf("validating response"))
			var attrGroup attributesgroup.AttributesGroup
			err = json.Unmarshal(body, &attrGroup)
			Expect(err).To(BeNil())
			Expect(attrGroup.GroupID).To(Equal(attrGroupID))
			Expect(len(attrGroup.Attributes)).To(Equal(len(rndAttributes)))
			for _, attr := range rndAttributes {
				Expect(attrGroup.Attributes).To(ContainElement(attr))
			}
		})

		It("POST smart attributes group returns 405 ", func() {

			By(fmt.Sprintf("Attempting to create attributes group in ReadOnlyMode"))
			description := fmt.Sprintf("Attributes group for test %s ", isolationID)
			reqBody := fmt.Sprintf(`{"description":"%s","attributes":["version","category"]}`, description)
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

		})
		It("DELETE smart attributes group returns 405", func() {

			By(fmt.Sprintf("Attempting to delete attributes group in ReadOnlyMode"))
			randomAttributes := []string{RandStringRunes(5), RandStringRunes(5), RandStringRunes(5)}
			attrGroupID1 := CreateAttributeGroup(ctx, database, endpointURISetup, isolationID, "", randomAttributes)

			uri := fmt.Sprintf("%s/%s", endpointURI, attrGroupID1)
			resp, body, err := HttpCallWithHeadersAndApiCallStat("DELETE", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

type ListAttributesGroupsResp struct {
	GroupID     string `json:"groupID" binding:"required"`
	Description string `json:"description,omitempty"`
}
