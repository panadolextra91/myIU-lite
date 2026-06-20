package grades

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ParsedScoreRow struct {
	RowIndex  int
	Username  string
	Score     float64
}

func ParseScoreCSV(r io.Reader) ([]ParsedScoreRow, []RowError) {
	reader := csv.NewReader(r)
	
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, []RowError{{Row: 0, Field: "file", Message: "file is empty"}}
		}
		return nil, []RowError{{Row: 0, Field: "file", Message: "failed to read csv"}}
	}

	if len(header) == 0 {
		return nil, []RowError{{Row: 0, Field: "header", Message: "missing header"}}
	}

	idIdx := -1
	scoreIdx := -1
	for i, h := range header {
		clean := strings.TrimSpace(strings.ToLower(h))
		if clean == "student_id" {
			idIdx = i
		} else if clean == "score" {
			scoreIdx = i
		}
	}

	var rowErrs []RowError

	if idIdx == -1 {
		rowErrs = append(rowErrs, RowError{Row: 1, Field: "header", Message: "missing 'student_id' column"})
	}
	if scoreIdx == -1 {
		rowErrs = append(rowErrs, RowError{Row: 1, Field: "header", Message: "missing 'score' column"})
	}
	if len(rowErrs) > 0 {
		return nil, rowErrs
	}

	var parsed []ParsedScoreRow
	seen := make(map[string]int)

	rowIndex := 2
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "row", Message: "malformed row"})
			rowIndex++
			continue
		}

		if len(record) <= idIdx {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "student_id", Message: "missing value"})
			rowIndex++
			continue
		}
		if len(record) <= scoreIdx {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "score", Message: "missing value"})
			rowIndex++
			continue
		}

		username := strings.TrimLeft(strings.TrimSpace(record[idIdx]), "=+-@\t\r ")
		if username == "" {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "student_id", Message: "empty value"})
			rowIndex++
			continue
		}

		scoreStr := strings.TrimLeft(strings.TrimSpace(record[scoreIdx]), "=+-@\t\r ")
		if scoreStr == "" {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "score", Message: "empty value"})
			rowIndex++
			continue
		}

		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "score", Message: "must be a number"})
			rowIndex++
			continue
		}
		if score < 0 || score > 100 {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "score", Message: "must be between 0 and 100"})
			rowIndex++
			continue
		}

		if prev, exists := seen[username]; exists {
			rowErrs = append(rowErrs, RowError{Row: rowIndex, Field: "student_id", Message: fmt.Sprintf("duplicate ID in file (matches row %d)", prev)})
			rowIndex++
			continue
		}

		seen[username] = rowIndex
		parsed = append(parsed, ParsedScoreRow{RowIndex: rowIndex, Username: username, Score: score})
		rowIndex++
	}

	return parsed, rowErrs
}
