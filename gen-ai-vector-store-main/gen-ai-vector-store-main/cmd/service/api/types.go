/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

type DeleteDocumentsResponse struct {
	DeletedDocuments int64 `json:"deletedDocuments"`
}

type ListAttributesGroupsItem struct {
	GroupID     string `json:"groupID"`
	Description string `json:"description"`
}
