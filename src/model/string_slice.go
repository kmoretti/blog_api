package model

import (
	"database/sql/driver"
	"encoding/json"
)

// StringSlice is a string slice that is stored as JSON text in the database.
// It tolerates invalid or NULL database values by falling back to an empty slice.
type StringSlice []string

// Scan implements sql.Scanner.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil
	}

	if len(data) == 0 {
		*s = StringSlice{}
		return nil
	}

	var parsed []string
	if err := json.Unmarshal(data, &parsed); err != nil {
		*s = StringSlice{}
		return nil
	}
	*s = StringSlice(parsed)
	return nil
}

// Value implements driver.Valuer.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	data, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
