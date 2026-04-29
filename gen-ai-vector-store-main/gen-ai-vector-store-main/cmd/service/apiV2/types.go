/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package apiV2

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/pagination"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
)

const (
	minIsolationIDLength  = 3
	minCollectionIDLength = 3
	maxIsolationIDLength  = 36
	maxCollectionIDLength = 255

	paramIsolationID  = "isolationID"
	paramCollectionID = "collectionID" // API v2 supports collectionID as URL encoded path parameter
	paramDocumentID   = "documentID"   // API v2 supports documentID URL as encoded path parameter
	serviceName       = "genai-vector-store"
)

type Collection struct {
	CollectionID string `json:"collectionID" binding:"required"`
}

type CollectionCreateRequest struct {
	CollectionID string `json:"collectionID" binding:"required"`
}

type ListCollectionsResponse struct {
	IsolationID string                   `json:"isolationID" binding:"required"`
	Collections []collections.Collection `json:"collections" binding:"required"`
	Pagination  pagination.Pagination    `json:"pagination,omitempty"`
}

type FindDocumentsRequestBody struct {
	Fields     []string              `json:"fields"`
	Filter     *FindDocumentsFilter  `json:"filter"`
	Pagination pagination.Pagination `json:"pagination"`
}

type FindDocumentsFilter struct {
	Attributes *AttributesFilterV2 `json:"attributes"`
	Status     string              `json:"status"`
}

type AttributesFilterV2 struct {
	Operator string                 `json:"operator" binding:"required"`
	Items    []AttributesFilterItem `json:"items" binding:"required"`
}

type AttributesFilterItem struct {
	Name     string   `json:"name" binding:"required"`
	Type     string   `json:"type"`
	Operator string   `json:"operator" binding:"required"`
	Values   []string `json:"values" binding:"required"`
}

func convertFilterToAttributes(filter *AttributesFilterV2) attributes.Filter {
	if filter == nil {
		return attributes.Filter{}
	}

	var attrFilter attributes.Filter
	attrFilter.Operator = filter.Operator
	for _, item := range filter.Items {
		attrFilter.Items = append(attrFilter.Items, attributes.AttributeFilter{
			Name:     item.Name,
			Type:     item.Type,
			Operator: item.Operator,
			Values:   item.Values,
		})
	}
	return attrFilter
}
