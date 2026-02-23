package postgres

import (
	"encoding/json"
	"fmt"
)

// DecodeRow decodes a single map[string]any row (from DaprDB.Query) into a
// struct T by round-tripping through JSON. T's fields must carry `json` tags
// that match the SQL column names returned by the query.
func DecodeRow[T any](row map[string]any) (T, error) {
	var v T
	b, err := json.Marshal(row)
	if err != nil {
		return v, fmt.Errorf("decode row: marshal: %w", err)
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("decode row: unmarshal: %w", err)
	}
	return v, nil
}

// DecodeRows decodes a slice of map[string]any rows into []T.
// Returns an empty (non-nil) slice when rows is empty.
func DecodeRows[T any](rows []map[string]any) ([]T, error) {
	if len(rows) == 0 {
		return []T{}, nil
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("decode rows: marshal: %w", err)
	}
	var result []T
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("decode rows: unmarshal: %w", err)
	}
	return result, nil
}
