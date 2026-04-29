/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package filters

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
)

func FilterAttributesToRetrieve(attrs []attributes.Attribute, retrieveAttributes []string) []attributes.Attribute {
	var result []attributes.Attribute
	for _, a := range attrs {
		if attributes.ContainsAttribute(retrieveAttributes, a) {
			result = append(result, a)
		}
	}
	return result
}
