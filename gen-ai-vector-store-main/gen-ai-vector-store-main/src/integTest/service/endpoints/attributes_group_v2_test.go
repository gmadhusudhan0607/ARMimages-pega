//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

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

var _ = Describe("Testing SVC /v1/*/smart-attributes-group ", func() {

	ctx := context.Background()
	var isolationID string
	var endpointURI string

	_ = Context("service is ready", func() {

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			endpointURI = fmt.Sprintf("%s/v1/%s/smart-attributes-group", baseURI, isolationID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsURI, isolationID)
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {

			//ag.GET("/", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.ListSmartAttributesGroups)
			ExpectServiceReturns404IfIsolationDoesNotExist("GET", endpointURI)

			//ag.POST("/", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.PostSmartAttributesGroup)
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)

			//ag.GET("/:groupID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.GetSmartAttributesGroup)
			ExpectServiceReturns404IfIsolationDoesNotExist("GET", fmt.Sprintf("%s/%s", endpointURI, "group-1"))

			//ag.DELETE("/:groupID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.DeleteSmartAttributesGroup)
			ExpectServiceReturns404IfIsolationDoesNotExist("DELETE", fmt.Sprintf("%s/%s", endpointURI, "group-1"))

		})

		It("can create attribute group", func() {
			By(fmt.Sprintf("Creating attributes group"))
			description := fmt.Sprintf("Attributes group for test %s ", isolationID)
			attrGroupID := CreateAttributeGroup(ctx, database, endpointURI, isolationID, description, []string{"version", "category"})
			By(fmt.Sprintf("Expect attributes group exists in DB"))
			ExpectAttributesGroupExistsInDB(ctx, database, isolationID, attrGroupID, description, []string{"version", "category"})
		})

		It("can list attribute group", func() {
			By(fmt.Sprintf("Creating attributes groups"))
			attrGroupID1 := CreateAttributeGroup(ctx, database, endpointURI, isolationID, "x", []string{RandStringRunes(5)})
			attrGroupID2 := CreateAttributeGroup(ctx, database, endpointURI, isolationID, "y", []string{RandStringRunes(5)})
			attrGroupID3 := CreateAttributeGroup(ctx, database, endpointURI, isolationID, "z", []string{RandStringRunes(5)})

			By(fmt.Sprintf("Calling attributes group endpoint"))
			resp, body, err := HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By(fmt.Sprintf("Validate list attributes groups endpoint returns all created groups"))
			var attrGroups []ListAttributesGroupsResp
			err = json.Unmarshal(body, &attrGroups)
			Expect(err).To(BeNil())
			Expect(len(attrGroups)).To(Equal(3))
			Expect(attrGroups).To(ContainElement(ListAttributesGroupsResp{GroupID: attrGroupID1, Description: "x"}))
			Expect(attrGroups).To(ContainElement(ListAttributesGroupsResp{GroupID: attrGroupID2, Description: "y"}))
			Expect(attrGroups).To(ContainElement(ListAttributesGroupsResp{GroupID: attrGroupID3, Description: "z"}))
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)
		})

		It("can get attribute group by ID", func() {
			By(fmt.Sprintf("Creating attributes group"))
			rndAttributes := []string{RandStringRunes(5), RandStringRunes(5), RandStringRunes(5)}
			description := fmt.Sprintf("Attributes group for test %s ", isolationID)
			attrGroupID := CreateAttributeGroup(ctx, database, endpointURI, isolationID, description, rndAttributes)

			By(fmt.Sprintf("Calling attributes group endpoint"))
			uri := fmt.Sprintf("%s/%s", endpointURI, attrGroupID)
			resp, body, err := HttpCall("GET", uri, nil, "")
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
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)
		})

		It("can delete attribute group", func() {
			By(fmt.Sprintf("Creating attributes group"))
			randomAttributes := []string{RandStringRunes(5), RandStringRunes(5), RandStringRunes(5)}
			attrGroupID1 := CreateAttributeGroup(ctx, database, endpointURI, isolationID, "", randomAttributes)

			By(fmt.Sprintf("Calling attributes group endpoint"))
			uri := fmt.Sprintf("%s/%s", endpointURI, attrGroupID1)
			resp, _, err := HttpCall("DELETE", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By(fmt.Sprintf("Expect attributes group does not exist in DB"))
			ExpectAttributesGroupDoesNotExistInDB(ctx, database, isolationID, attrGroupID1)

			By(fmt.Sprintf("Calling attributes group endpoint again should return 200"))
			resp, _, err = HttpCall("DELETE", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

	})
})

type ListAttributesGroupsResp struct {
	GroupID     string `json:"groupID" binding:"required"`
	Description string `json:"description,omitempty"`
}
