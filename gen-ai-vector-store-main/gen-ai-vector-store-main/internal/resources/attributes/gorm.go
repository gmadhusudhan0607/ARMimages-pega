/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Values implements Valuer Interface:
func (a *AttrValues) Value() (driver.Value, error) {
	// Serialize the Address struct into a format suitable for storage
	// For example, you might serialize it into a JSON string
	objJSON, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(objJSON), nil
}

// Scan implements Scanner Interface:
func (a *AttrValues) Scan(src interface{}) error {
	// Deserialize the value from the database into the Address struct
	// For example, if the value is stored as a JSON string, you would unmarshal it
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	case AttrValues:
		*a = src.(AttrValues)
		return nil
	default:
		return fmt.Errorf("AttrValues.Scan: unsupported data type: %T, value: %v", src, src)
	}
	return json.Unmarshal(data, a)
}

// Values implements Valuer Interface:
func (a *Attributes) Value() (driver.Value, error) {
	// Serialize the Address struct into a format suitable for storage
	// For example, you might serialize it into a JSON string
	objJSON, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(objJSON), nil
}

func (a *Attributes) Scan(src interface{}) error {
	// Deserialize the value from the database into the Address struct
	// For example, if the value is stored as a JSON string, you would unmarshal it
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	case Attributes:
		*a = append(*a, src.(Attributes)...)
		return nil
	default:
		return fmt.Errorf("Attributes.Scan: unsupported data type: %T, value: %v", src, src)
	}
	return json.Unmarshal(data, a)
}

func (a *Attribute) Scan(src interface{}) error {
	// Deserialize the value from the database into the Address struct
	// For example, if the value is stored as a JSON string, you would unmarshal it
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	default:
		return fmt.Errorf("Attribute.Scan: unsupported data type: %T, value: %v", src, src)
	}
	return json.Unmarshal(data, a)
}

// Values implements Valuer Interface for AttributesV2:
func (a *AttributesV2) Value() (driver.Value, error) {
	objJSON, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(objJSON), nil
}

// Scan implements Scanner Interface for AttributesV2:
func (a *AttributesV2) Scan(src interface{}) error {
	// Handle NULL values from database
	if src == nil {
		*a = make(AttributesV2)
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	case AttributesV2:
		*a = src.(AttributesV2)
		return nil
	default:
		return fmt.Errorf("AttributesV2.Scan: unsupported data type: %T, value: %v", src, src)
	}
	return json.Unmarshal(data, a)
}
