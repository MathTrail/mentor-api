package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// scanRows reads all rows from a pgx.Rows into a slice of maps keyed by column
// name. It closes the rows and checks for iteration errors before returning.
func scanRows(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	fieldDescs := rows.FieldDescriptions()
	var result []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("pgxpool scan row: %w", err)
		}
		row := make(map[string]any, len(fieldDescs))
		for i, fd := range fieldDescs {
			row[fd.Name] = values[i]
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgxpool rows: %w", err)
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// DecodeRow decodes a single map[string]any row (from DB.Query) into a
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
