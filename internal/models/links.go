package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type Link struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Links is a JSONB-backed list of profile links.
type Links []Link

// Value implements driver.Valuer so Links marshals to JSONB on write.
func (l Links) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("models.Links.Value: %w", err)
	}
	return b, nil
}

// Scan implements sql.Scanner so JSONB columns decode into Links.
func (l *Links) Scan(src any) error {
	if src == nil {
		*l = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("models.Links.Scan: unsupported source type")
	}
	if len(b) == 0 {
		*l = nil
		return nil
	}
	if err := json.Unmarshal(b, l); err != nil {
		return fmt.Errorf("models.Links.Scan: %w", err)
	}
	return nil
}
