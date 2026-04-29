/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributesgroup

type AttributesGroup struct {
	GroupID     string   `json:"groupID"`
	Description string   `json:"description,omitempty"`
	Attributes  []string `json:"attributes,omitempty"`
}
