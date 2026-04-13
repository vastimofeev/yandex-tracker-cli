package model

import (
	"bytes"
	"encoding/json"
)

type StringID string

func (s *StringID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*s = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*s = StringID(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*s = StringID(number.String())
		return nil
	}

	return nil
}
